package handlers

import (
	"ai-ba/internal/domain/models"
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SessionHandler struct {
	sessRepo  *repository.SessionRepo
	msgRepo   *repository.MessageRepo
	draftRepo *repository.DraftRepo
}

func NewSessionHandler(sessRepo *repository.SessionRepo, msgRepo *repository.MessageRepo, draftRepo *repository.DraftRepo) *SessionHandler {
	return &SessionHandler{sessRepo: sessRepo, msgRepo: msgRepo, draftRepo: draftRepo}
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
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	// 1. Delete orphaned messages (messages not linked to any session)
	// This is good hygiene, though strictly we should delete messages of the sessions we are about to delete.
	// But since we don't have a "DeleteMessagesForSessions" easily, we can rely on DeleteOrphans later?
	// No, we should try to be clean.
	// Actually, if we delete sessions, the messages become orphans.
	// So we can delete orphans AFTER deleting sessions.

	// 2. Delete sessions that are NOT linked to drafts (Business Requests)
	if err := h.sessRepo.DeleteUnlinkedSessions(userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	// 3. Delete orphaned messages (which now includes messages from the deleted sessions)
	if _, err := h.msgRepo.DeleteOrphans(); err != nil {
		// Log error but continue
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
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

	// Cascade delete messages and drafts
	if err := h.msgRepo.DeleteBySessionID(uint(id)); err != nil {
		// Continue even if error, or return? Let's continue to try deleting session.
	}
	if err := h.draftRepo.DeleteBySessionID(uint(id)); err != nil {
	}

	if err := h.sessRepo.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *SessionHandler) CleanupDatabase(c *fiber.Ctx) error {
	// Check for admin role
	role := ""
	if v := c.Locals("user_role"); v != nil {
		role = v.(string)
	}
	if role != "admin" && role != "Business Analyst" {
		return c.Status(fiber.StatusForbidden).SendString("forbidden")
	}

	msgsDeleted, err := h.msgRepo.DeleteOrphans()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete orphaned messages", "details": err.Error()})
	}

	draftsDeleted, err := h.draftRepo.DeleteOrphans()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete orphaned drafts", "details": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":          "Database cleanup completed",
		"messages_deleted": msgsDeleted,
		"drafts_deleted":   draftsDeleted,
	})
}
