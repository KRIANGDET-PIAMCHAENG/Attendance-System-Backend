package repository

import (
	//"context"
	"fmt"
	//"my-app/internal/entity"
	"time"
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
	RoleID    string `json:"role-id"`    // เปลี่ยนเป็น role-id
	RoleName  string `json:"role-name"`  // เปลี่ยนเป็น role-name
	RoleColor string `json:"role-color"` // เปลี่ยนเป็น role-color
}

type UserInfo struct {
	// ใส่ gorm column tag ให้ครบทุกฟิลด์เพื่อความชัวร์
	UserID       string `json:"user_id" gorm:"column:user_id"`
	EmployeeID   string `json:"employee_id" gorm:"column:employee_id"`
	Email        string `json:"email" gorm:"column:email"`
	FullNameThai string `json:"fullname_thai" gorm:"column:fullname_thai"`
	FullNameEng  string `json:"fullname_eng" gorm:"column:fullname_eng"`
	Gender       string `json:"gender" gorm:"column:gender"`
	Nationality  string `json:"nationality" gorm:"column:nationality"`
	Phone        string `json:"phone" gorm:"column:phone"`
	RoleInit     string `json:"role_init" gorm:"column:role_init"`
	Picture      string `json:"picture" gorm:"column:picture"`

	// เปลี่ยน JSON Key จาก role_sys เป็น roles ตามที่สั่ง
	Roles []UserRole `json:"roles" gorm:"-"` 
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

type UpdateUserRequest struct {
	NameTH      string `json:"name-th"`
	NameEN      string `json:"name-en"`
	EmployeeID  string `json:"employee-id"`
	Gender      string `json:"gender"`
	Nationality string `json:"nationality"`
	Phone       string `json:"phone"`
	InitialRole string `json:"initial-role"` // รับมาเพื่อให้ Bind JSON ผ่าน แต่จะไม่เอาไป Update
}

type UpdateRoleRequest struct {
	ID    string `json:"id" binding:"required"`
	Name  string `json:"name"`
	Type  string `json:"type"` // เพิ่ม type เข้ามา
	Color string `json:"color"`
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserInfo(id string) (*UserInfo, error) {
	var info UserInfo
	
	// ใช้ GORM Method API แทน Raw SQL เพื่อการ Mapping ที่แม่นยำกว่า
	err := r.db.Table("user_info").Where("user_id = ?", id).Take(&info).Error
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var roles []UserRole
	// ดึง Role ID, Name, Color มาพร้อมกัน
	roleQuery := `
        SELECT r.role_id, r.role_name, r.role_color
        FROM role r
        JOIN user_roles ur ON r.role_id = ur.role_id
        WHERE ur.user_id = ?
    `
	r.db.Raw(roleQuery, id).Scan(&roles)

	// สร้าง Initial Role (Hardcode ID เป็น 000)
	initRole := UserRole{
		RoleID:    "000",
		RoleName:  info.RoleInit,
		RoleColor: "808080",
	}

	// เอา Initial Role ไว้หน้าสุด และตามด้วย Roles อื่นๆ
	info.Roles = append([]UserRole{initRole}, roles...)

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

func (r *UserRepo) UpdateUserInfo(id string, req UpdateUserRequest) error {
	// ใช้ Map เพื่อเลือกเฉพาะฟิลด์ที่ต้องการอัปเดต (ตัด role_init, email, picture ออก)
	result := r.db.Table("user_info").Where("user_id = ?", id).Updates(map[string]interface{}{
		"fullname_thai": req.NameTH,
		"fullname_eng":  req.NameEN,
		"employee_id":   req.EmployeeID,
		"gender":        req.Gender,
		"nationality":   req.Nationality,
		"phone":         req.Phone,
		// ไม่มีการอัปเดต email, picture และ role_init
	})

	if result.Error != nil {
		return result.Error
	}
    
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found with id: %s", id)
	}

	return nil
}
func (r *UserRepo) UpdateRole(req UpdateRoleRequest) error {
	// อัปเดตตาราง role โดยระบุ role_id
	// GORM จะ Update เฉพาะฟิลด์ที่ระบุใน Map
	result := r.db.Table("role").Where("role_id = ?", req.ID).Updates(map[string]interface{}{
		"role_name":  req.Name,
		"role_type":  req.Type, // อัปเดตประเภท (main/special)
		"role_color": req.Color,
	})

	if result.Error != nil {
		return result.Error
	}

	// เช็คว่ามีแถวถูกแก้จริงไหม
	if result.RowsAffected == 0 {
		return fmt.Errorf("role not found with id: %s", req.ID)
	}

	return nil
}


type MaxLeavePart struct {
	Sick      float64 `json:"sick"`
	Personal  float64 `json:"personal"`
	Vacation  float64 `json:"vacation"`
	Maternity float64 `json:"maternity"`
	Paternity float64 `json:"paternity"`
	Parental  float64 `json:"parental"`
}

type UserInfoPart struct {
	NameTH      string `json:"name-th"`
	NameEN      string `json:"name-en"`
	EmployeeID  string `json:"employee-id"`
	Gender      string `json:"gender"`
	Nationality string `json:"nationality"`
	Phone       string `json:"phone"`
	InitialRole string `json:"initial-role"`
}

type CreateUserFullRequest struct {
	ID       string       `json:"id"`
	Email    string       `json:"email"`
	UserInfo UserInfoPart `json:"user-info"`
	MaxLeave MaxLeavePart `json:"max-leave"`
}

type UpdateUserRolesRequest struct {
	Roles []string `json:"roles"` // ["001", "002"]
}

// --- [NEW] Functions ---

// 1. Create User (Full Transaction: Users + Info + Role + Leave Balances)
func (r *UserRepo) CreateUserFull(req CreateUserFullRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A. Insert Users (Master)
		if err := tx.Table("users").Create(map[string]interface{}{
			"user_id":         req.ID,
			"employee_id":     req.UserInfo.EmployeeID,
			"manager_role_id": "2", // Default (เช่น ผู้ดูแล)
		}).Error; err != nil {
			return err
		}

		// B. Insert UserInfo
		if err := tx.Table("user_info").Create(map[string]interface{}{
			"user_id":       req.ID,
			"employee_id":   req.UserInfo.EmployeeID,
			"email":         req.Email,
			"fullname_thai": req.UserInfo.NameTH,
			"fullname_eng":  req.UserInfo.NameEN,
			"gender":        req.UserInfo.Gender,
			"nationality":   req.UserInfo.Nationality,
			"phone":         req.UserInfo.Phone,
			"role_init":     req.UserInfo.InitialRole,
			"picture":       "", 
		}).Error; err != nil {
			return err
		}

		// C. Insert Default Role (Role ID 3 = บุคคลทั่วไป ตามข้อมูลเก่า)
		// if err := tx.Table("user_roles").Create(map[string]interface{}{
		// 	"user_id": req.ID,
		// 	"role_id": "3", 
		// }).Error; err != nil {
		// 	return err
		// }

		// D. Insert Max Leave (Map JSON key -> DB leave_types)
		leaveMap := map[string]float64{
			"sick":      req.MaxLeave.Sick,
			"personal":  req.MaxLeave.Personal,
			"vacation":  req.MaxLeave.Vacation,
			"maternity": req.MaxLeave.Maternity,
			"paternity": req.MaxLeave.Paternity,
			"parental":  req.MaxLeave.Parental,
		}

		// ใช้ปีปัจจุบันเป็นหลัก
		currentYear := time.Now().Year() 

		for nameEn, days := range leaveMap {
			var typeID int
			// หา ID จากตาราง leave_types โดยใช้ชื่อภาษาอังกฤษ (name_en)
			// LOWER() เพื่อป้องกัน case sensitive (เช่น Sick vs sick)
			if err := tx.Table("leave_types").Select("id").Where("LOWER(name_en) = LOWER(?)", nameEn).Scan(&typeID).Error; err == nil && typeID != 0 {
				
				// Insert ลง leave_balances
				if err := tx.Table("leave_balances").Create(map[string]interface{}{
					"user_id":       req.ID,
					"leave_type_id": typeID,
					"year":          currentYear,
					"days_allowed":  days,
					"days_used":     0,
				}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// 2. Update Roles (Replace Strategy: ลบของเก่า -> ใส่ของใหม่)
func (r *UserRepo) UpdateUserRoles(userID string, roleIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A. ลบ Role เก่าทั้งหมดของ User นี้ทิ้งก่อน
		if err := tx.Table("user_roles").Where("user_id = ?", userID).Delete(nil).Error; err != nil {
			return err
		}

		// B. วนลูปใส่ Role ใหม่เข้าไป
		for _, rid := range roleIDs {
			if err := tx.Table("user_roles").Create(map[string]interface{}{
				"user_id": userID,
				"role_id": rid,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *UserRepo) UpdateUserMaxLeave(userID string, req MaxLeavePart) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		leaveMap := map[string]float64{
			"sick":      req.Sick,
			"personal":  req.Personal,
			"vacation":  req.Vacation,
			"maternity": req.Maternity,
			"paternity": req.Paternity,
			"parental":  req.Parental,
		}

		for nameEn, days := range leaveMap {
			var typeID int
			// 1. หา typeID จากชื่อเหมือนเดิม
			err := tx.Table("leave_types").Select("id").Where("LOWER(name_en) = LOWER(?)", nameEn).Scan(&typeID).Error
			
			if err == nil && typeID != 0 {
				// 2. อัปเดตโดยเช็คแค่ user_id และ leave_type_id 
				// เพื่อให้มันไปทับข้อมูลเดิม (ID 1-6) ที่ไม่มีค่า year ได้
				result := tx.Table("leave_balances").
					Where("user_id = ? AND leave_type_id = ?", userID, typeID).
					Update("days_allowed", days)
				
				// 3. ถ้าไม่มีข้อมูลอยู่เลยจริงๆ ค่อยสร้างใหม่ (Optional)
				// แต่ถ้าคุณอยากให้ "Update อย่างเดียว" สามารถลบ block if ด้านล่างนี้ทิ้งได้เลยครับ
				if result.RowsAffected == 0 {
					tx.Table("leave_balances").Create(map[string]interface{}{
						"user_id":       userID,
						"leave_type_id": typeID,
						"days_allowed":  days,
						"days_used":     0,
						"year":          2026,
					})
				}
			}
		}
		return nil
	})
}