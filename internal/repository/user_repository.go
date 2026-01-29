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

type UserInfoLogin struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Role    string `json:"role"`
}

type UserInfo struct {
	UserID       string `json:"user_id"`
	EmployeeID   string `json:"employee_id"`
	Email        string `json:"email"`
	FullNameThai string `json:"fullname_thai"`
	FullNameEng  string `json:"fullname_eng"`
	Gender       string `json:"gender"`
	Nationality  string `json:"nationality"`
	Phone 	     string `json:"phone"`
	RoleInit     string `json:"role_init"`
	// เราอาจจะตัด Nationality หรือ Phone ออกถ้าหน้านั้นไม่ต้องโชว์
	// เพื่อความปลอดภัยตามหลัก Privacy by Design
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserInfo(id string) (*UserInfo, error) {
    var info UserInfo

    // 🚩 ระบุชื่อคอลัมน์ให้ตรงกับ struct เพื่อความปลอดภัยและประสิทธิภาพ
    query := `
        SELECT 
            user_id, employee_id, email, fullname_thai, fullname_eng, 
            gender, nationality, phone, role_init 
        FROM user_info 
        WHERE user_id = ?
    `
    result := r.db.Raw(query, id).Scan(&info)

    if result.Error != nil {
        return nil, result.Error
    }
    if result.RowsAffected == 0 {
        return nil, fmt.Errorf("user not found with id: %s", id)
    }
    return &info, nil
}

// func (r *UserRepo) GetRoleByEmail(email string) (string, error) {
//     var roleType string

//     query := `  SELECT r.role_type
//                 FROM user_info ui
//                 JOIN user_roles ur ON ui.user_id = ur.user_id
//                 JOIN role r ON ur.role_id = r.role_id
//                 WHERE LOWER(ui.email) = LOWER(?)    -- ใช้ ? แทน $1 ถ้าใช้ผ่าน GORM Raw
//                 LIMIT 1
//              `

//     // GORM จะเอาค่า email ไปแทนที่ ? ให้เอง และป้องกัน SQL Injection ให้ด้วย
//     result := r.db.Raw(query, email).Scan(&roleType)

//     if result.Error != nil {
//         return "", result.Error
//     }

//     if result.RowsAffected == 0 {
//         // คืนค่าว่างกลับไปเพื่อให้ Handler รู้ว่าหาไม่เจอ
//         return "", fmt.Errorf("role not found for email: %s", email)
//     }

//     return roleType, nil
// }

func (r *UserRepo) GetUserInfoByEmail(email string) (*UserInfoLogin, error) {
    var info UserInfoLogin

    // 🚩 ต้องใช้ fullname_eng AS name (หรือ fullname_thai แล้วแต่จะเลือก)
    // เพื่อให้ GORM Scan เข้าฟิลด์ Name ใน Struct ได้
    query := `
        SELECT ui.user_id, ui.fullname_eng AS name, ui.email, r.role_type AS role
        FROM user_info ui
        JOIN user_roles ur ON ui.user_id = ur.user_id
        JOIN role r ON ur.role_id = r.role_id
        WHERE LOWER(ui.email) = LOWER(?)
        LIMIT 1
    `

    result := r.db.Raw(query, email).Scan(&info)

    if result.Error != nil {
        return nil, result.Error
    }

    if result.RowsAffected == 0 {
        return nil, fmt.Errorf("user not found with email: %s", email)
    }

    return &info, nil
}