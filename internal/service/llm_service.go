package service

import (
	"ai-ba/internal/domain/models"
	"ai-ba/internal/prompts"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type LLMService struct {
	apiKey string
}

func NewLLMService() *LLMService {
	return &LLMService{
		apiKey: os.Getenv("AI_KEY"),
	}
}

func (s *LLMService) Generate(prompt string) (string, error) {
	if s.apiKey == "" {
		return "", errors.New("Google API key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(s.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	// --- ИСПРАВЛЕННЫЙ БЛОК SAFETY ---
	model.SafetySettings = []*genai.SafetySetting{
		{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockNone},
	}
	// ---------------------------------

	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(1000000)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("Gemini API request failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", errors.New("no candidates in response")
	}

	// Проверка причины остановки (для отладки)
	if resp.Candidates[0].FinishReason != genai.FinishReasonStop {
		fmt.Printf("stopped with reason: %s", resp.Candidates[0].FinishReason)
		return "", fmt.Errorf("stopped with reason: %s", resp.Candidates[0].FinishReason)
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			return string(txt), nil
		}
	}
	fmt.Println("response did not contain text")
	return "", errors.New("response did not contain text")
}

func (s *LLMService) AnalyzeRequest(userReq string) (*AnalysisData, error) {
	prompt := fmt.Sprintf(prompts.AnalysisPromptTemplate, userReq)

	jsonStr, err := s.Generate(prompt)
	if err != nil {
		return nil, err
	}

	// Clean up potential markdown code blocks if the LLM adds them
	jsonStr = cleanJSON(jsonStr)

	var data AnalysisData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nResponse: %s", err, jsonStr)
	}

	return &data, nil
}

// Chat generates a response based on the conversation history.
// It injects a system prompt to guide the AI behavior.
func (s *LLMService) Chat(history []models.Message, userInput string) (string, error) {
	if s.apiKey == "" {
		return "", errors.New("Google API key is not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(s.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()
	model := client.GenerativeModel("gemini-2.5-flash")
	model.SetTemperature(0.7)

	// System Prompt
	systemPrompt := prompts.ChatSystemPrompt

	cs := model.StartChat()
	cs.History = []*genai.Content{
		{
			Role: "user",
			Parts: []genai.Part{
				genai.Text(systemPrompt),
			},
		},
		{
			Role: "model",
			Parts: []genai.Part{
				genai.Text("Understood. I am ready to interview the user."),
			},
		},
	}

	// Helper to add message to history, combining if same role
	addMsg := func(role, text string) {
		if len(cs.History) > 0 {
			last := cs.History[len(cs.History)-1]
			if last.Role == role {
				// Combine with previous message
				if txt, ok := last.Parts[0].(genai.Text); ok {
					last.Parts[0] = genai.Text(string(txt) + "\n" + text)
				}
				return
			}
		}
		cs.History = append(cs.History, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(text)},
		})
	}

	for _, m := range history {
		role := "user"
		if m.Author == "ai" {
			role = "model"
		}
		addMsg(role, m.Text)
	}

	resp, err := cs.SendMessage(ctx, genai.Text(userInput))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 {
		return "", errors.New("no candidates")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			return string(txt), nil
		}
	}
	return "", errors.New("no text in response")
}

// ExtractDataFromChat analyzes the full conversation history to produce the JSON report.
func (s *LLMService) ExtractDataFromChat(history []string) (*AnalysisData, error) {
	// Join history into a single transcript
	transcript := strings.Join(history, "\n")

	prompt := fmt.Sprintf(prompts.TranscriptAnalysisPromptTemplate, transcript)

	return s.AnalyzeRequest(prompt) // Reuse the existing parsing logic
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}
