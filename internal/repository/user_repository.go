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
	RoleGen  string   `json:"-" gorm:"column:role"`
    Role []string `json:"role"`
    Picture      string `json:"picture"`
}

type UserRole struct {
    RoleName string `json:"role_name"`
    RoleColor string `json:"role_color"`
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
    Picture      string `json:"picture"`
	// เราอาจจะตัด Nationality หรือ Phone ออกถ้าหน้านั้นไม่ต้องโชว์
	// เพื่อความปลอดภัยตามหลัก Privacy by Design

    Roles        []UserRole `json:"role_sys"`
}

type InitInfoResponse struct {
    UserID    string   `json:"user_id"`
    Name      string   `json:"name"`
    Picture   string   `json:"picture"`
    AllRoles  []string `json:"all_roles"`
    RoleInit  string   `json:"role_init"`
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserInfo(id string) (*UserInfo, error) {
    var info UserInfo
    var roles []UserRole

    // 1. ดึงข้อมูล User หลัก (เหมือนเดิม)
    query := `
        SELECT user_id, employee_id, email, fullname_thai, fullname_eng, 
               gender, nationality, phone, role_init, picture
        FROM user_info 
        WHERE user_id = ?
    `
    result := r.db.Raw(query, id).Scan(&info)
    if result.Error != nil { return nil, result.Error }
    if result.RowsAffected == 0 { return nil, fmt.Errorf("user not found: %s", id) }

    // 2. ดึงข้อมูล Roles (ลบ Query ที่ผิดออก แล้วใช้ตัวที่ Join ถูกต้อง)
    roleQuery := `
        SELECT r.role_name, r.role_color
        FROM role r
        JOIN user_roles ur ON r.role_id = ur.role_id
        WHERE ur.user_id = ?
    `
    r.db.Raw(roleQuery, id).Scan(&roles)

    // 🚩 3. Hard code เพิ่ม Role จาก RoleInit เข้าไป
    // สร้าง Object ใหม่โดยใช้ค่าจาก info.RoleInit และใส่สีเทา (#808080)
    initRole := UserRole{
        RoleName: info.RoleInit,
        RoleColor: "#808080",  
    }

    // ยัดเข้าเข้าไปใน Slice (ในตัวอย่างนี้เอาไว้ลำดับแรกสุด)
    roles = append([]UserRole{initRole}, roles...)

    // 4. รวมร่างข้อมูล
    info.Roles = roles

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

func (r *UserRepo) AllRole(id string) ([]string, error) {
    var roleNames []string

    // 🚩 SQL Join: ดึงเฉพาะ role_name จากตาราง role
    // โดยเชื่อมผ่านตารางกลาง user_roles ด้วย user_id
    query := `
        SELECT r.role_type
        FROM role r
        JOIN user_roles ur ON r.role_id = ur.role_id
        WHERE ur.user_id = ?
    `

    // ใช้ Pluck เพื่อดึงข้อมูลคอลัมน์เดียวออกมาใส่ใน Slice
    result := r.db.Raw(query, id).Pluck("role_type", &roleNames)

    if result.Error != nil {
        return nil, result.Error
    }

    return roleNames, nil
}

func (r *UserRepo) GetUserInfoByEmail(email string) (*UserInfoLogin, error) {
    var info UserInfoLogin

    // 🚩 ต้องใช้ fullname_eng AS name (หรือ fullname_thai แล้วแต่จะเลือก)
    // เพื่อให้ GORM Scan เข้าฟิลด์ Name ใน Struct ได้
    query := `
        SELECT ui.user_id, ui.fullname_eng AS name, ui.email , r.role_type AS role_gen, picture
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

    role, err := r.AllRole(info.UserID)
    if err != nil {
        return nil, err
    }

    info.Role = role
    return &info, nil
}

// UpdatePicture ใช้สำหรับอัปเดต URL รูปภาพของ User โดยอิงจาก Email
func (r *UserRepo) UpdatePicture(email string, pictureURL string) error {
    // ใช้ Exec เพื่อรัน Raw SQL ในการ Update
    query := `UPDATE user_info SET picture = ? WHERE LOWER(email) = LOWER(?)`
    
    result := r.db.Exec(query, pictureURL, email)
    
    if result.Error != nil {
        return result.Error
    }
    
    // ตรวจสอบว่ามีการอัปเดตจริงไหม (เผื่อไม่มี email นี้ในระบบ)
    if result.RowsAffected == 0 {
        return fmt.Errorf("no user found with email: %s", email)
    }
    
    return nil
}

func (r *UserRepo) GetInitInfo(id string) (*InitInfoResponse, error) {
    var res InitInfoResponse
    
    // ดึงข้อมูลพื้นฐานจาก user_info
    query := `SELECT user_id, fullname_eng AS name, picture FROM user_info WHERE user_id = ?`
    err := r.db.Raw(query, id).Scan(&res).Error
    if err != nil {
        return nil, err
    }

    // ดึง Roles ทั้งหมด (เรียกใช้ AllRole ที่เราเขียนไว้แล้ว)
    roles, err := r.AllRole(id)
    if err != nil {
        return nil, err
    }
    res.AllRoles = roles

    return &res, nil
}