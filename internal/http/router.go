package http

import (
	"ai-ba/internal/http/handlers"
	"ai-ba/internal/http/middleware"
	"ai-ba/internal/repository"
	"ai-ba/internal/service"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

// NewRouter теперь принимает дополнительные зависимости для ws
func NewRouter(authService *service.AuthService, llm *service.LLMService, msgRepo *repository.MessageRepo, sessRepo *repository.SessionRepo, draftRepo *repository.DraftRepo, teamMsgRepo *repository.TeamMessageRepo, userRepo *repository.UserRepo, hub *Hub, jwtSecret string) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	authHandler := handlers.NewAuthHandler(authService)

	// Public auth endpoints
	// Public auth endpoints
	// Registration disabled for public
	// app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)
	app.Post("/auth/google", authHandler.LoginWithGoogle)

	// Admin endpoints
	app.Post("/api/admin/users", middleware.AuthMiddleware(jwtSecret), authHandler.CreateUser)

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Protected example endpoint
	app.Get("/me", middleware.AuthMiddleware(jwtSecret), func(c *fiber.Ctx) error {
		// Construct user from claims
		user := map[string]interface{}{
			"id":    middleware.GetUserID(c),
			"email": strings.TrimPrefix(middleware.GetUser(c), "local:"), // simple hack, better to store email separately
			"name":  middleware.GetUserName(c),
			"role":  middleware.GetUserRole(c),
		}
		return c.JSON(fiber.Map{"user": user})
	})

	// List users for Team Chat
	app.Get("/api/users", middleware.AuthMiddleware(jwtSecret), func(c *fiber.Ctx) error {
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

	sessionHandler := handlers.NewSessionHandler(sessRepo, msgRepo)
	app.Get("/sessions", middleware.AuthMiddleware(jwtSecret), sessionHandler.GetSessions)
	app.Post("/sessions", middleware.AuthMiddleware(jwtSecret), sessionHandler.CreateSession)
	app.Delete("/sessions", middleware.AuthMiddleware(jwtSecret), sessionHandler.ClearSessions)
	app.Delete("/sessions/:id", middleware.AuthMiddleware(jwtSecret), sessionHandler.DeleteSession)
	app.Get("/sessions/:id/messages", middleware.AuthMiddleware(jwtSecret), sessionHandler.GetMessages)

	draftHandler := handlers.NewDraftHandler(draftRepo)
	app.Get("/drafts", middleware.AuthMiddleware(jwtSecret), draftHandler.GetDrafts)
	app.Delete("/drafts", middleware.AuthMiddleware(jwtSecret), draftHandler.ClearDrafts)
	app.Delete("/drafts/:id", middleware.AuthMiddleware(jwtSecret), draftHandler.DeleteDraft)

	// WebSocket endpoint (agent)
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			authHeader := c.Get("Authorization")
			queryToken := c.Query("token")
			c.Locals("authHeader", authHeader)
			c.Locals("queryToken", queryToken)
			return c.Next()
		}
		return c.Next()
	})

	app.Get("/ws/agent", websocket.New(
		NewWSHandler(llm, msgRepo, sessRepo, draftRepo, jwtSecret),
	))

	// WebSocket endpoint (team)
	app.Get("/ws/team", websocket.New(
		NewTeamWSHandler(hub, teamMsgRepo, jwtSecret),
	))

	return app
}
