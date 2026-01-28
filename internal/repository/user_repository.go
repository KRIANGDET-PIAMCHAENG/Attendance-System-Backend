package repository

import (
	//"context"
	"fmt"
	//"my-app/internal/entity"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetRoleByEmail(email string) (string, error) {
	var roleType string

	query := `  SELECT r.role_type
            FROM user_info ui
            JOIN user_roles ur ON ui.user_id = ur.user_id
            JOIN role r ON ur.role_id = r.role_id
            WHERE ui.email = $1
            LIMIT 1
         `

	result := r.db.Raw(query, email).Scan(&roleType)

	if result.Error != nil {
		return "", result.Error
	}

	if result.RowsAffected == 0 {
		return "" , fmt.Errorf("role not found email : %s", email)
	}

	return roleType, nil
}
