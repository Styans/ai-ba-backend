package models

import "time"

type TeamMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   uint      `json:"sender_id"`
	ReceiverID uint      `json:"receiver_id"` // 0 for broadcast/public channel if we had one, but for DM it's specific
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}
