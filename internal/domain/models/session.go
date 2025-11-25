package models

type Session struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	UserID    uint  `json:"user_id"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
	// ...existing code...
}
