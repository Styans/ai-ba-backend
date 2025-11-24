package http

import (
	"ai-ba/internal/http/handlers"
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/service"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func NewRouter(authService *service.AuthService) *fiber.App {
	app := fiber.New()

	authHandler := handlers.NewAuthHandler(authService)

	// Public auth endpoints
	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)
	app.Post("/auth/google", authHandler.LoginWithGoogle)

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Protected example endpoint
	app.Get("/me", middleware.AuthMiddleware(), func(c *fiber.Ctx) error {
		user := middleware.GetUser(c)
		fmt.Println("===================================================--------")
		fmt.Println(c)
		fmt.Println("===================================================--------")
		return c.JSON(fiber.Map{"user": user})
	})

	return app
}
