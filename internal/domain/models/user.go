package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`        // "User", "Business Analyst", "Admin"
	Position     string    `json:"position"`    // New field: e.g. "Senior Manager", "Developer"
	Provider     string    `json:"provider"`    // "local" или "google"
	ProviderID   string    `json:"provider_id"` // id from provider
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
