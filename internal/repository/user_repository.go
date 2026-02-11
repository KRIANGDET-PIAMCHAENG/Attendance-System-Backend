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
	UserID  string   `json:"user_id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	RoleGen string   `json:"-" gorm:"column:role"`
	Role    []string `json:"role"`
	Picture string   `json:"picture"`
}

type UserRole struct {
	RoleName  string `json:"role_name"`
	RoleColor string `json:"role_color"`
}

type UserInfo struct {
	UserID       string `json:"user_id"`
	EmployeeID   string `json:"employee_id"`
	Email        string `json:"email"`
	FullNameThai string `json:"fullname_thai" gorm:"column:fullname_thai"`

	// 🚩 เติม gorm:"column:fullname_eng"
	FullNameEng string `json:"fullname_eng"  gorm:"column:fullname_eng"`
	Gender      string `json:"gender"`
	Nationality string `json:"nationality"`
	Phone       string `json:"phone"`
	RoleInit    string `json:"role_init"`
	Picture     string `json:"picture"`
	// เราอาจจะตัด Nationality หรือ Phone ออกถ้าหน้านั้นไม่ต้องโชว์
	// เพื่อความปลอดภัยตามหลัก Privacy by Design

	Roles []UserRole `json:"role_sys" gorm:"-"`
}

type InitInfoResponse struct {
	UserID   string   `json:"user_id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	AllRoles []string `json:"all_roles"`
	// RoleInit  string   `json:"role_init"`
	Picture string `json:"picture"`
}

type RoleAPI struct {
	RoleID    string `json:"role-id"`
	RoleName  string `json:"role-name"`
	RoleColor string `json:"role-color"`
}
type UserAPI struct {
    ID          string    `json:"id"`
    NameTH      string    `json:"name-th"`
    NameEN      string    `json:"name-en"`
    AvatarURL   string    `json:"avatar-url"`
    EmployeeID  string    `json:"employee-id"`
    Gender      string    `json:"gender"`
    Nationality string    `json:"nationality"`
    Phone       string    `json:"phone"`
    Email       string    `json:"email"`
    InitialRole string    `json:"initial-role"`
    
    // 🚩 เติม gorm:"-" ต่อท้ายแบบนี้ครับ
    Roles       []RoleAPI `json:"roles" gorm:"-"` 
}

type RoleMemberAPI struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Color  string `json:"color"`
	Member int    `json:"member"` // ตัวนี้จะเก็บค่าจากการ COUNT
}

type LeaveQuotaResult struct {
    TypeKey     string  `gorm:"column:name_en"`    // เช่น sick, personal
    DaysAllowed float64 `gorm:"column:days_allowed"` // เช่น 60, 45
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserInfo(id string) (*UserInfo, error) {

	fmt.Printf("--- Debug: Repository looking for ID: [%s] (Length: %d) ---\n", id, len(id))
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
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("user not found: %s", id)
	}

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
		RoleName:  info.RoleInit,
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
	query := `SELECT user_id, email, fullname_eng AS name, picture FROM user_info WHERE user_id = ?`
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

func (r *UserRepo) GetAllUsers() ([]UserAPI, error) {
	var users []UserAPI

	// 1. ดึงข้อมูล User ทั้งหมดจากตาราง user_info
	// แมปคอลัมน์ DB ให้ตรงกับ Struct UserAPI
	query := `
		SELECT 
			user_id as id,
			fullname_thai as name_th,
			fullname_eng as name_en,
			picture as avatar_url,
			employee_id,
			gender,
			nationality,
			phone,
			email,
			role_init as initial_role
		FROM user_info
	`
	
	// ใช้ Scan เข้า Struct โดย GORM จะจับคู่ชื่อ field ให้ (หรือเรา alias ใน SQL ให้ตรงก็ได้)
	result := r.db.Raw(query).Scan(&users)
	if result.Error != nil {
		return nil, result.Error
	}

	// 2. วนลูปเพื่อดึง Roles ของแต่ละ User (วิธีนี้ง่ายสุดและชัวร์เรื่อง Data)
	for i := range users {
		var roles []RoleAPI
		roleQuery := `
			SELECT r.role_id, r.role_name, r.role_color
			FROM role r
			JOIN user_roles ur ON r.role_id = ur.role_id
			WHERE ur.user_id = ?
		`
		// ดึง Role และใส่เข้าไปใน User คนนั้นๆ
		r.db.Raw(roleQuery, users[i].ID).Scan(&roles)
		
		// ถ้าไม่มี Role ให้เป็น Array ว่างแทน null (เพื่อความสวยงามของ JSON)
		if roles == nil {
			roles = []RoleAPI{}
		}
		users[i].Roles = roles
	}

	return users, nil
}

func (r *UserRepo) GetAllRoles() ([]RoleMemberAPI, error) {
	var roles []RoleMemberAPI

	// SQL Query:
	// 1. เลือกข้อมูล Role
	// 2. ใช้ LEFT JOIN กับ user_roles เพื่อดึงคนที่ถือ Role นี้
	// 3. ใช้ COUNT(ur.user_id) เพื่อนับจำนวนคน
	// 4. GROUP BY เพื่อรวมกลุ่มตาม Role
	query := `
		SELECT 
			r.role_id AS id,
			r.role_name AS name,
			r.role_type AS type,
			r.role_color AS color,
			COUNT(ur.user_id) AS member
		FROM role r
		LEFT JOIN user_roles ur ON r.role_id = ur.role_id
		GROUP BY r.role_id, r.role_name, r.role_type, r.role_color
		ORDER BY r.role_id
	`
	
	result := r.db.Raw(query).Scan(&roles)
	if result.Error != nil {
		return nil, result.Error
	}

	return roles, nil
}


func (r *UserRepo) GetLeaveQuotas(userID string) ([]LeaveQuotaResult, error) {
    var results []LeaveQuotaResult

    // Join ตาราง balances กับ types เพื่อเอาชื่อภาษาอังกฤษ (name_en) มาเป็น Key
    query := `
        SELECT lt.name_en, lb.days_allowed
        FROM leave_balances lb
        JOIN leave_types lt ON lb.leave_type_id = lt.id
        WHERE lb.user_id = ?
    `

    err := r.db.Raw(query, userID).Scan(&results).Error
    if err != nil {
        return nil, err
    }

    return results, nil
}