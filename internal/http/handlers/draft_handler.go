package handlers

import (
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type DraftHandler struct {
	repo *repository.DraftRepo
}

func NewDraftHandler(repo *repository.DraftRepo) *DraftHandler {
	return &DraftHandler{repo: repo}
}

func (h *DraftHandler) GetDrafts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	drafts, err := h.repo.ListByUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list drafts"})
	}

	return c.JSON(drafts)
}
func (h *DraftHandler) ClearDrafts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	fmt.Printf("DEBUG: ClearDrafts called for userID: %d\n", userID)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	if err := h.repo.DeleteAllByUser(userID); err != nil {
		fmt.Printf("DEBUG: Error deleting drafts: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	fmt.Println("DEBUG: Drafts deleted successfully")

	return c.SendStatus(fiber.StatusOK)
}

func (h *DraftHandler) DeleteDraft(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid draft id")
	}

	if err := h.repo.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
