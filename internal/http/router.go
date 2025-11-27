package http

import (
	"ai-ba/internal/http/handlers"
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"
	"ai-ba/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// NewRouter теперь принимает дополнительные зависимости для ws и drafts
func NewRouter(
	authService *service.AuthService,
	llm *service.LLMService,
	draftService *service.DraftService,
	msgRepo *repository.MessageRepo,
	sessRepo *repository.SessionRepo,
) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3001",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	authHandler := handlers.NewAuthHandler(authService)
	draftHandler := handlers.NewDraftHandler(draftService)

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
		return c.JSON(fiber.Map{"user": user})
	})

	// Drafts endpoints (Protected)
	drafts := app.Group("/drafts", middleware.AuthMiddleware())
	drafts.Post("/", draftHandler.Create)
	drafts.Get("/", draftHandler.List)
	drafts.Get("/:id/download", draftHandler.Download)
	drafts.Post("/:id/approve", draftHandler.Approve)

	// WebSocket endpoint (agent). Клиент должен подключаться к /ws/agent?token=<jwt>
	// Если запрос не является WebSocket-upgrade — возвращаем понятный JSON вместо дефолтного 426.
	app.Use("/ws/agent", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// забираем токен из заголовка и/или query
			authHeader := c.Get("Authorization") // "Bearer xxx"
			queryToken := c.Query("token")       // ?token=xxx

			// сохраняем в Locals, они попадут в websocket.Conn
			c.Locals("authHeader", authHeader)
			c.Locals("queryToken", queryToken)

			return c.Next()
		}

		c.Status(fiber.StatusUpgradeRequired)
		return c.JSON(fiber.Map{
			"error":   "Upgrade Required",
			"message": "This endpoint requires a WebSocket upgrade. Connect with ws:// or wss://...",
		})
	})

	// 2) Сам WS-обработчик
	app.Get("/ws/agent", websocket.New(
		NewWSHandler(llm, draftService, msgRepo, sessRepo),
	))

	return app
}
