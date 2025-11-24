package app

import (
	"fmt"
	"log"
	"os"

	"ai-ba/internal/config"
	router "ai-ba/internal/http"
	"ai-ba/internal/repository"
	"ai-ba/internal/service"

	"github.com/joho/godotenv"
)

func Run() error {
	// Попробовать загрузить .env (если есть)
	_ = godotenv.Load()

	// Загрузить конфиг (yaml + env override)
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return err
	}

	// DSN из конфигурации (env имеет приоритет)
	dsn := cfg.Database.DSN
	if dsn == "" {
		return fmt.Errorf("DB_DSN not set")
	}

	db, err := repository.ConnectDB(dsn)
	if err != nil {
		return err
	}

	// repos
	userRepo := repository.NewUserRepo(db)
	// other repos can be created as needed (session/message/draft/review)

	// services
	authService := service.NewAuthService(userRepo)

	// router
	app := router.NewRouter(authService)

	port := cfg.Server.Port
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Println("starting server on", addr)
	return app.Listen(addr)
}
