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
	draftRepo *repository.DraftRepo,
	teamMsgRepo *repository.TeamMessageRepo,
	userRepo *repository.UserRepo,
	hub *Hub,
	jwtSecret string,
) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	authHandler := handlers.NewAuthHandler(authService)
	draftHandler := handlers.NewDraftHandler(draftService)
	sessionHandler := handlers.NewSessionHandler(sessRepo, msgRepo)

	// Public auth endpoints
	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)
	app.Post("/auth/google", authHandler.LoginWithGoogle)

	// Admin endpoints
	app.Post("/api/admin/users", middleware.AuthMiddleware(), authHandler.CreateUser)

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Protected example endpoint
	app.Get("/me", middleware.AuthMiddleware(), func(c *fiber.Ctx) error {
		user := middleware.GetUser(c)
		return c.JSON(fiber.Map{"user": user})
	})

	// List users for Team Chat
	app.Get("/api/users", middleware.AuthMiddleware(), func(c *fiber.Ctx) error {
		currentUserID := middleware.GetUserID(c)
		users, err := userRepo.GetAll()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch users"})
		}

		// Filter out current user
		otherUsers := make([]map[string]interface{}, 0)
		for _, u := range users {
			if u.ID != currentUserID {
				otherUsers = append(otherUsers, map[string]interface{}{
					"id":    u.ID,
					"name":  u.Name,
					"email": u.Email,
				})
			}
		}
		return c.JSON(otherUsers)
	})

	// Session endpoints
	app.Get("/sessions", middleware.AuthMiddleware(), sessionHandler.GetSessions)
	app.Post("/sessions", middleware.AuthMiddleware(), sessionHandler.CreateSession)
	app.Delete("/sessions", middleware.AuthMiddleware(), sessionHandler.ClearSessions)
	app.Delete("/sessions/:id", middleware.AuthMiddleware(), sessionHandler.DeleteSession)
	app.Get("/sessions/:id/messages", middleware.AuthMiddleware(), sessionHandler.GetMessages)

	// Drafts endpoints (Protected)
	drafts := app.Group("/drafts", middleware.AuthMiddleware())
	drafts.Post("/", draftHandler.Create)
	drafts.Get("/", draftHandler.List)
	drafts.Delete("/", draftHandler.ClearDrafts)
	drafts.Delete("/:id", draftHandler.DeleteDraft)
	drafts.Get("/:id/download", draftHandler.Download)
	drafts.Post("/:id/approve", draftHandler.Approve)

	// WebSocket endpoint (agent). Клиент должен подключаться к /ws/agent?token=<jwt>
	app.Use("/ws/agent", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			authHeader := c.Get("Authorization")
			queryToken := c.Query("token")
			c.Locals("authHeader", authHeader)
			c.Locals("queryToken", queryToken)
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
			"error":   "Upgrade Required",
			"message": "This endpoint requires a WebSocket upgrade.",
		})
	})

	// 2) Сам WS-обработчик
	app.Get("/ws/agent", websocket.New(
		NewWSHandler(llm, draftService, msgRepo, sessRepo),
	))

	// WebSocket endpoint (team)
	app.Get("/ws/team", websocket.New(
		NewTeamWSHandler(hub, teamMsgRepo, jwtSecret),
	))

	return app
}
