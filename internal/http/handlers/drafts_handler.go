package handlers

import (
	"ai-ba/internal/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type DraftHandler struct {
	service *service.DraftService
}

func NewDraftHandler(s *service.DraftService) *DraftHandler {
	return &DraftHandler{service: s}
}

func (h *DraftHandler) Create(c *fiber.Ctx) error {
	type req struct {
		Title   string `json:"title"`
		Request string `json:"request"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	draft, err := h.service.CreateFromRequest(r.Title, r.Request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(draft)
}

func (h *DraftHandler) List(c *fiber.Ctx) error {
	drafts, err := h.service.List(100)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(drafts)
}

func (h *DraftHandler) Download(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
	}

	draft, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Draft not found")
	}

	return c.Download(draft.FilePath)
}

func (h *DraftHandler) Approve(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
	}

	if err := h.service.Approve(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "approved"})
}
