package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type DraftRepo struct {
	db *gorm.DB
}

func NewDraftRepo(db *gorm.DB) *DraftRepo { return &DraftRepo{db: db} }

func (r *DraftRepo) Create(d *models.Draft) error {
	return r.db.Create(d).Error
}

func (r *DraftRepo) GetByID(id uint) (*models.Draft, error) {
	var d models.Draft
	if err := r.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DraftRepo) List(offset, limit int) ([]models.Draft, error) {
	var list []models.Draft
	q := r.db.Order("id DESC")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DraftRepo) Update(d *models.Draft) error {
	return r.db.Save(d).Error
}

func (r *DraftRepo) Delete(id uint) error {
	return r.db.Delete(&models.Draft{}, id).Error
}

func (r *DraftRepo) ListByUser(userID uint) ([]models.Draft, error) {
	var list []models.Draft
	if err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *DraftRepo) DeleteAllByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Draft{}).Error
}

func (r *DraftRepo) GetBySessionID(sessionID uint) (*models.Draft, error) {
	var d models.Draft
	if err := r.db.Where("session_id = ?", sessionID).Order("created_at DESC").First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}
