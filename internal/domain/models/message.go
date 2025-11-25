package models

type Message struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	SessionID uint   `json:"session_id"`
	Author    string `json:"author"` // "user" или "ai"
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
