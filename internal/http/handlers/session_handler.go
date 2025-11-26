package handlers

import (
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type SessionHandler struct {
	sessRepo *repository.SessionRepo
}

func NewSessionHandler(sessRepo *repository.SessionRepo) *SessionHandler {
	return &SessionHandler{sessRepo: sessRepo}
}

func (h *SessionHandler) GetSessions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	sessions, err := h.sessRepo.ListByUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"sessions": sessions})
}
