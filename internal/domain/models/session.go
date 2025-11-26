package models

type Session struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `json:"user_id"`
	Title     string `json:"title"`
	Status    string `json:"status"` // reviewing, accepted, rejected
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	// ...existing code...
}
