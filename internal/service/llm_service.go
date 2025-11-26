package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type LLMService struct {
	apiKey string
}

func NewLLMService(apiKey string) *LLMService {
	return &LLMService{
		apiKey: apiKey,
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
		fmt.Printf("Gemini API error: %v\n", err)
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
