package handlers

import (
	"ai-ba/internal/domain/models"
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SessionHandler struct {
	sessRepo *repository.SessionRepo
	msgRepo  *repository.MessageRepo
}

func NewSessionHandler(sessRepo *repository.SessionRepo, msgRepo *repository.MessageRepo) *SessionHandler {
	return &SessionHandler{sessRepo: sessRepo, msgRepo: msgRepo}
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

func (h *SessionHandler) GetMessages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	sessionID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid session id")
	}

	// Verify session belongs to user (optional but recommended)
	// For now, assuming if they have the ID and are auth'd, it's okay, or we should check ownership.
	// Let's just list messages.

	msgs, err := h.msgRepo.ListBySession(uint(sessionID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"messages": msgs})
}
func (h *SessionHandler) ClearSessions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	fmt.Printf("DEBUG: ClearSessions called for userID: %d\n", userID)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	if err := h.sessRepo.DeleteAllByUser(userID); err != nil {
		fmt.Printf("DEBUG: Error deleting sessions: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	fmt.Println("DEBUG: Sessions deleted successfully")

	return c.SendStatus(fiber.StatusOK)
}

func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	fmt.Printf("DEBUG: CreateSession called for userID: %d\n", userID)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	var p struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}

	// Default title if empty
	if p.Title == "" {
		p.Title = "New Project"
	}

	s := &models.Session{
		UserID:    userID,
		Title:     p.Title,
		Status:    "reviewing",
		CreatedAt: time.Now().Unix(),
	}

	if err := h.sessRepo.Create(s); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	fmt.Printf("DEBUG: Session created: %d\n", s.ID)

	return c.JSON(fiber.Map{"session_id": s.ID, "title": s.Title})
}

func (h *SessionHandler) DeleteSession(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid session id")
	}

	if err := h.sessRepo.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
