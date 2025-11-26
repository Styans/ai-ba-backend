package models

import (
	"time"
)

type Draft struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Title             string    `json:"title"`
	Content           string    `json:"content"` // Raw text or summary
	Status            string    `json:"status"`  // PENDING, APPROVED, REJECTED
	FilePath          string    `json:"file_path"`
	StructuredContent []byte    `gorm:"type:jsonb" json:"structured_content"` // JSON data for doc generation
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
