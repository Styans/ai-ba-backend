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

func (r *MessageRepo) GetBySessionID(sessionID uint) ([]models.Message, error) {
	return r.ListBySession(sessionID)
}

func (r *MessageRepo) DeleteBySessionID(sessionID uint) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&models.Message{}).Error
}

func (r *MessageRepo) DeleteAllByUser(userID uint) error {
	// Messages don't have user_id directly, they are linked via session_id.
	// So we need to delete messages where session_id IN (SELECT id FROM sessions WHERE user_id = ?)
	// OR, we can just rely on the fact that we are deleting sessions right after.
	// But to be clean, let's do a join delete or subquery.
	// GORM: db.Where("session_id IN (?)", db.Table("sessions").Select("id").Where("user_id = ?", userID)).Delete(&models.Message{})
	return r.db.Where("session_id IN (?)", r.db.Table("sessions").Select("id").Where("user_id = ?", userID)).Delete(&models.Message{}).Error
}

func (r *MessageRepo) DeleteOrphans() (int64, error) {
	result := r.db.Where("session_id NOT IN (?)", r.db.Table("sessions").Select("id")).Delete(&models.Message{})
	return result.RowsAffected, result.Error
}
