package repository

import (
	"ai-ba/internal/domain/models"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) FindByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(u *models.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepo) FindByProvider(provider, providerID string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) UpsertByProvider(provider, providerID string, u *models.User) (*models.User, error) {
	existing, err := r.FindByProvider(provider, providerID)
	if err == nil {
		// update fields
		existing.Name = u.Name
		existing.Email = u.Email
		if err := r.db.Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	if err == gorm.ErrRecordNotFound {
		if err := r.db.Create(u).Error; err != nil {
			return nil, err
		}
		return u, nil
	}
	return nil, err
}

func (r *UserRepo) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	return users, err
}
