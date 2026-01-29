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
                WHERE LOWER(ui.email) = LOWER(?)    -- ใช้ ? แทน $1 ถ้าใช้ผ่าน GORM Raw
                LIMIT 1
             `

    // GORM จะเอาค่า email ไปแทนที่ ? ให้เอง และป้องกัน SQL Injection ให้ด้วย
    result := r.db.Raw(query, email).Scan(&roleType)

    if result.Error != nil {
        return "", result.Error
    }

    if result.RowsAffected == 0 {
        // คืนค่าว่างกลับไปเพื่อให้ Handler รู้ว่าหาไม่เจอ
        return "", fmt.Errorf("role not found for email: %s", email)
    }

    return roleType, nil
}
