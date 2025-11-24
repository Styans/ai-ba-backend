package ai

import "ai-ba/internal/domain/models"

func MapToDraft(aiText string) *models.Draft {
	// TODO: парсинг ответа AI в Draft/US/BRD структуры
	return &models.Draft{
		Title:   "mapped title",
		Content: aiText,
	}
}
