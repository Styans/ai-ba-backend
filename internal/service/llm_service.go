package service

import (
	"ai-ba/internal/domain/models"
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
	prompt := fmt.Sprintf(`
You are an expert Business Analyst. Analyze the following user request and generate a structured Business Analysis Report.
Return ONLY valid JSON (no markdown formatting, no backticks) with the following structure:
{
	"goal": "Goal statement formulated according to SMART criteria (Specific, Measurable, Achievable, Relevant, Time-bound)",
	"description": "Detailed description",
	"scope": "In/Out of scope",
	"business_rules": ["Rule 1", "Rule 2"],
	"kpis": ["KPI 1", "KPI 2"],
	"use_cases": ["Use Case 1", "Use Case 2"],
	"user_stories": ["As a... I want to... So that..."],
	"diagrams_desc": ["Description of a flowchart", "Description of a sequence diagram"]
}

User Request: %s
`, userReq)

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
	systemPrompt := `
You are an intelligent business assistant. Your goal is to gather requirements via a strict 5-stage questionnaire.
After the user's initial request, you must start the process.

STAGES (5 questions each):
1. Goal of the Request
2. Target Audience & User Roles
3. Business Process & Constraints
4. Expected Results & KPIs
5. Technical Requirements & Integrations

PROTOCOL:
1. Receive user request.
2. Send Stage 1 questions (JSON).
3. Wait for answers.
4. Send Stage 2 questions (JSON).
...
5. After Stage 5 answers, send Final Requirements (JSON).

OUTPUT FORMATS (Strict JSON ONLY, no markdown, no other text):

TYPE 1: QUESTIONNAIRE (Stages 1-5)
{
  "type": "questionnaire",
  "stage": <1-5>,
  "title": "<Stage Title>",
  "questions": [
    { "id": "q1", "text": "Question 1" },
    { "id": "q2", "text": "Question 2" },
    { "id": "q3", "text": "Question 3" },
    { "id": "q4", "text": "Question 4" },
    { "id": "q5", "text": "Question 5" }
  ]
}

TYPE 2: REQUIREMENTS (Final)
{
  "type": "requirements",
  "smart_requirements": {
    "specific": "...",
    "measurable": "...",
    "achievable": "...",
    "relevant": "...",
    "time_bound": "..."
  },
  "summary": "Brief task description",
  "answers": [
    { "step": 1, "question": "...", "answer": "..." },
    ...
  ]
}

RULES:
- Do NOT output markdown code blocks (like ` + "`" + `json ... ` + "`" + `). Output raw JSON string.
- Do NOT add any conversational text (e.g., "Here is the next stage"). ONLY JSON.
- Wait for the user to answer ALL questions of the current stage before moving to the next.
`

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

	prompt := fmt.Sprintf(`
Analyze the following conversation transcript between a User and a Business Analyst.
Extract all requirements and generate a structured Business Analysis Report.
Return ONLY valid JSON (no markdown) with this structure:
{
	"goal": "...",
	"description": "...",
	"scope": "...",
	"business_rules": [...],
	"kpis": [...],
	"use_cases": [...],
	"user_stories": [...],
	"diagrams_desc": [...]
}

Transcript:
%s
`, transcript)

	return s.AnalyzeRequest(prompt) // Reuse the existing parsing logic
}

func (s *LLMService) AnalyzeSmartRequest(prompt string) (*SmartAnalysisData, error) {
	jsonStr, err := s.Generate(prompt)
	if err != nil {
		return nil, err
	}

	jsonStr = cleanJSON(jsonStr)

	var data SmartAnalysisData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nResponse: %s", err, jsonStr)
	}

	return &data, nil
}

// ExtractSmartDataFromChat analyzes the full conversation history to produce the JSON report.
func (s *LLMService) ExtractSmartDataFromChat(history []string) (*SmartAnalysisData, error) {
	transcript := strings.Join(history, "\n")

	prompt := fmt.Sprintf(`
Analyze the following conversation transcript between a User and a Business Analyst.
Extract all requirements and generate a structured Business Analysis Report.
Return ONLY valid JSON (no markdown) with this structure:
{
  "questions": [
    {"step": 1, "question": "...", "answer": "..."},
    ...
  ],
  "smart_requirements": {
    "specific": "...",
    "measurable": "...",
    "achievable": "...",
    "relevant": "...",
    "time_bound": "..."
  },
  "summary": "Short task description"
}

Transcript:
%s
`, transcript)

	return s.AnalyzeSmartRequest(prompt)
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}
