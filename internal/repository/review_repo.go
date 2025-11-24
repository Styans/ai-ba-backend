package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type ReviewRepo struct {
	db *gorm.DB
}

func NewReviewRepo(db *gorm.DB) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) Save(rv *models.Review) error {
	return r.db.Create(rv).Error
}

func (r *ReviewRepo) GetByID(id uint) (*models.Review, error) {
	var rv models.Review
	if err := r.db.First(&rv, id).Error; err != nil {
		return nil, err
	}
	return &rv, nil
}

func (r *ReviewRepo) ListByDraft(draftID uint) ([]models.Review, error) {
	var list []models.Review
	if err := r.db.Where("draft_id = ?", draftID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
