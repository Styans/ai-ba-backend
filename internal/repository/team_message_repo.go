package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type TeamMessageRepo struct {
	db *gorm.DB
}

func NewTeamMessageRepo(db *gorm.DB) *TeamMessageRepo {
	return &TeamMessageRepo{db: db}
}

func (r *TeamMessageRepo) Save(msg *models.TeamMessage) error {
	return r.db.Create(msg).Error
}

// GetHistory returns messages between two users
func (r *TeamMessageRepo) GetHistory(user1ID, user2ID uint) ([]models.TeamMessage, error) {
	var messages []models.TeamMessage
	// Select messages where (sender=u1 AND receiver=u2) OR (sender=u2 AND receiver=u1)
	err := r.db.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		user1ID, user2ID, user2ID, user1ID,
	).Order("created_at asc").Find(&messages).Error
	return messages, err
}
