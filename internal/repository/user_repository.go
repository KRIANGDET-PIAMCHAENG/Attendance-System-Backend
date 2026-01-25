package repository

import (
	"my-app/internal/entity"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindByEmail(email string) (*entity.UserInfo, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// ฟังก์ชันค้นหา User จาก Email
func (r *userRepository) FindByEmail(email string) (*entity.UserInfo, error) {
	var user entity.UserInfo
	// ค้นหาในตาราง UserInfo โดยดู column email
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}