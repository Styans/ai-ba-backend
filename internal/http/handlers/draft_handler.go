package handlers

import (
	"ai-ba/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type DraftHandler struct {
	repo *repository.DraftRepo
}

func NewDraftHandler(repo *repository.DraftRepo) *DraftHandler {
	return &DraftHandler{repo: repo}
}

func (h *DraftHandler) GetDrafts(c *fiber.Ctx) error {
	// В реальном приложении стоит фильтровать по userID
	// userID := middleware.GetUserID(c)

	// Пока просто возвращаем все (или можно добавить метод ListByUser в репозиторий)
	drafts, err := h.repo.List(0, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list drafts"})
	}

	return c.JSON(drafts)
}
