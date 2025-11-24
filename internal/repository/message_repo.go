package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo { return &MessageRepo{db: db} }

func (r *MessageRepo) Save(m *models.Message) error {
	return r.db.Create(m).Error
}

func (r *MessageRepo) GetByID(id uint) (*models.Message, error) {
	var m models.Message
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MessageRepo) ListBySession(sessionID uint) ([]models.Message, error) {
	var msgs []models.Message
	if err := r.db.Where("session_id = ?", sessionID).Order("id ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}
