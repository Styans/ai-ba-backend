package handlers

import (
	"ai-ba/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(a *service.AuthService) *AuthHandler { return &AuthHandler{auth: a} }

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
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

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("bad request")
	}
	tok, err := h.auth.Register(req.Email, req.Password, req.Name)
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
