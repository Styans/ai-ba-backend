package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectDB устанавливает соединение с базой данных Postgres и выполняет авто-миграцию моделей.
func ConnectDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Авто‑миграция основных моделей
	if err := db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Message{},
		&models.Draft{},
		&models.Review{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
