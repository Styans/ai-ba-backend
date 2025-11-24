package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type SessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(s *models.Session) error {
	return r.db.Create(s).Error
}

func (r *SessionRepo) GetByID(id uint) (*models.Session, error) {
	var s models.Session
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) ListByUser(userID uint) ([]models.Session, error) {
	var list []models.Session
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
