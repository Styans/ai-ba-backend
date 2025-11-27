package handlers

import (
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(a *service.AuthService) *AuthHandler { return &AuthHandler{auth: a} }

type createUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type googleReq struct {
	IDToken string `json:"id_token"`
}

type tokenResp struct {
	Token string `json:"token"`
}

func (h *AuthHandler) CreateUser(c *fiber.Ctx) error {
	// Check role
	role := middleware.GetUserRole(c)
	if role != "Business Analyst" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied"})
	}

	var req createUserReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("bad request")
	}

	// Default role if not provided? Or require it?
	if req.Role == "" {
		req.Role = "User"
	}

	tok, err := h.auth.CreateUser(req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return c.JSON(tokenResp{Token: tok})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("bad request")
	}
	tok, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}
	return c.JSON(tokenResp{Token: tok})
}

func (h *AuthHandler) LoginWithGoogle(c *fiber.Ctx) error {
	var req googleReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("bad request")
	}
	tok, err := h.auth.LoginWithGoogle(c.Context(), req.IDToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}
	return c.JSON(tokenResp{Token: tok})
}
