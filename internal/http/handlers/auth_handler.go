package handlers

import (
	"ai-ba/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var p struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}

	token, err := h.service.Register(p.Email, p.Password, p.Name)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"token": token})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var p struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}

	token, err := h.service.Login(p.Email, p.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"token": token})
}

func (h *AuthHandler) LoginWithGoogle(c *fiber.Ctx) error {
	var p struct {
		IDToken string `json:"idToken"`
	}
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}

	token, err := h.service.LoginWithGoogle(c.Context(), p.IDToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"token": token})
}

func (h *AuthHandler) CreateUser(c *fiber.Ctx) error {
	// Admin only? Middleware handles auth, but maybe check role here?
	// For now just implement the call.

	var p struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Position string `json:"position"`
	}
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}

	token, err := h.service.CreateUser(p.Email, p.Password, p.Name, p.Role, p.Position)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"token": token, "message": "user created"})
}
