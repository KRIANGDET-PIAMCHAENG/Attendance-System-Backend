package repository

import (
	//"context"
	"fmt"
	//"my-app/internal/entity"
	"gorm.io/gorm"
	"time"
)

type UserRepo struct {
	db *gorm.DB
}

// TableName ระบุชื่อตารางให้ชัดเจนว่าเป็น "role" (ไม่ใช่ roles)
func (Role) TableName() string {
	return "role"
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

    // 🚩 แก้ตรงนี้: เอา JOIN ออก ดึงแค่ข้อมูลจากตาราง user_info ก็พอ
    query := `
        SELECT user_id, fullname_eng AS name, email, picture
        FROM user_info
        WHERE LOWER(email) = LOWER(?)
        LIMIT 1
    `

    result := r.db.Raw(query, email).Scan(&info)

    if result.Error != nil {
        return nil, result.Error
    }

    if result.RowsAffected == 0 {
        return nil, fmt.Errorf("user not found with email: %s", email)
    }

    // 🚩 ไปดึง Role แยกตรงนี้แทน (ถ้า User ไม่มี Role ฟังก์ชันนี้ควรคืนค่าเป็น Slice เปล่า ๆ และไม่ Error)
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

	// 1. ดึง User ทั้งหมด (Query ที่ 1)
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
	if err := r.db.Raw(query).Scan(&users).Error; err != nil {
		return nil, err
	}

	// ถ้าไม่มี User เลย ก็จบ
	if len(users) == 0 {
		return []UserAPI{}, nil
	}

	// 2. เก็บ UserID ทั้งหมดใส่ Map เพื่อรอจับคู่
	userMap := make(map[string]*UserAPI)
	var userIDs []string
	for i := range users {
		users[i].Roles = []RoleAPI{} // กัน nil
		userMap[users[i].ID] = &users[i]
		userIDs = append(userIDs, users[i].ID)
	}

	// 3. ดึง Role ของ "ทุกคนในรายการ" มาในครั้งเดียว (Query ที่ 2)
	// ใช้ WHERE IN (...) แทนการวนลูป
	type UserRoleResult struct {
		UserID    string `json:"user_id"`
		RoleID    string `json:"role-id"`
		RoleName  string `json:"role-name"`
		RoleColor string `json:"role-color"`
	}

	var roleResults []UserRoleResult
	roleQuery := `
        SELECT ur.user_id, r.role_id, r.role_name, r.role_color
        FROM role r
        JOIN user_roles ur ON r.role_id = ur.role_id
        WHERE ur.user_id IN ? 
    `
	// GORM จะแปลง slice userIDs เป็น a,b,c ให้เอง
	if err := r.db.Raw(roleQuery, userIDs).Scan(&roleResults).Error; err != nil {
		return nil, err
	}

	// 4. จับคู่ Role ใส่ User ใน Memory (เร็วมาก)
	for _, res := range roleResults {
		if user, exists := userMap[res.UserID]; exists {
			user.Roles = append(user.Roles, RoleAPI{
				RoleID:    res.RoleID,
				RoleName:  res.RoleName,
				RoleColor: res.RoleColor,
			})
		}
	}

	return users, nil
}

func (r *UserRepo) GetAllRoles() ([]RoleMemberAPI, error) {
	var roles []RoleMemberAPI

	// SQL Query:
	// 1. เลือกข้อมูล Role หลัก (id, name, type, color)
	// 2. ใช้ LEFT JOIN กับตาราง subordinate_manager_roles (smr) เพื่อดึงลูกน้อง
	// 3. ใช้ COUNT(smr.subordinate_id) เพื่อนับจำนวนลูกน้องในสังกัด
	// 4. GROUP BY เพื่อรวมกลุ่มตาม Role
	query := `
		SELECT 
			r.role_id AS id,
			r.role_name AS name,
			r.role_type AS type,
			r.role_color AS color,
			COUNT(smr.subordinate_id) AS member
		FROM role r
		LEFT JOIN subordinate_manager_roles smr ON r.role_id = smr.manager_role_id
		GROUP BY r.role_id, r.role_name, r.role_type, r.role_color
		ORDER BY r.role_id
	`

	// ใช้ Raw SQL เพื่อความชัวร์และรวดเร็ว
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
	// ใช้ Map เพื่อเลือกเฉพาะฟิลด์ที่ต้องการอัปเดต
	result := r.db.Table("user_info").Where("user_id = ?", id).Updates(map[string]interface{}{
		"fullname_thai": req.NameTH,
		"fullname_eng":  req.NameEN,
		"employee_id":   req.EmployeeID,
		"gender":        req.Gender,
		"nationality":   req.Nationality,
		"phone":         req.Phone,

		// 🚩 [NEW] ปลดล็อกบรรทัดนี้เพื่อให้ Update Role Init ได้
		"role_init": req.InitialRole,

		// (ส่วน Email และ Picture เรายังคงไม่ให้แก้ผ่าน API นี้ เพื่อความปลอดภัย)
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

// 1. Create User (Full Transaction: Users + Info + Role + Leave Balances)
func (r *UserRepo) CreateUserFull(req CreateUserFullRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A. Insert Users (Master)
		if err := tx.Table("users").Create(map[string]interface{}{
			"user_id":     req.ID,
			"employee_id": req.UserInfo.EmployeeID,
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
		currentYear, err := GetBudgetYear(tx, time.Now())
		if err != nil {
			return err // คืนค่า Error ถ้าดึง Config ไม่ผ่าน
		}

		for nameEn, days := range leaveMap {
			var typeID int
			// หา ID จากตาราง leave_types ...
			if err := tx.Table("leave_types").Select("id").Where("LOWER(name_en) = LOWER(?)", nameEn).Scan(&typeID).Error; err == nil && typeID != 0 {

				// Insert ลง leave_balances
				if err := tx.Table("leave_balances").Create(map[string]interface{}{
					"user_id":       req.ID,
					"leave_type_id": typeID,
					"year":          currentYear, // ✅ ค่าปีงบประมาณที่ถูกต้องจะลง Database ตรงนี้
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

// 🚩 เปลี่ยน []int เป็น []string
func (r *UserRepo) UpdateUserRoles(userID string, roleIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. ลบของเดิม
		if err := tx.Table("user_roles").Where("user_id = ?", userID).Delete(nil).Error; err != nil {
			return err
		}

		// 2. ใส่ของใหม่
		for _, roleID := range roleIDs {
			data := map[string]interface{}{
				"user_id":    userID,
				"role_id":    roleID, // ส่งเป็น string ไปเลย DB จะรับได้พอดี
				"created_at": time.Now(),
			}
			if err := tx.Table("user_roles").Create(data).Error; err != nil {
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

// ... (code เดิม)

// DeleteUser ลบข้อมูลพนักงานและข้อมูลที่เกี่ยวข้องทั้งหมด
func (r *UserRepo) DeleteUser(userID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. ลบวันลาคงเหลือ (Child Table)
		if err := tx.Table("leave_balances").Where("user_id = ?", userID).Delete(nil).Error; err != nil {
			return err
		}

		// 2. ลบสิทธิ์การใช้งาน (Child Table)
		if err := tx.Table("user_roles").Where("user_id = ?", userID).Delete(nil).Error; err != nil {
			return err
		}

		// 3. ลบข้อมูลส่วนตัว (Child Table)
		if err := tx.Table("user_info").Where("user_id = ?", userID).Delete(nil).Error; err != nil {
			return err
		}

		// 4. ลบ User หลัก (Parent Table)
		result := tx.Table("users").Where("user_id = ?", userID).Delete(nil)

		if result.Error != nil {
			return result.Error
		}

		// เช็คว่ามีข้อมูลถูกลบจริงไหม (ถ้าไม่มีแสดงว่า ID ผิด)
		if result.RowsAffected == 0 {
			return fmt.Errorf("user not found with id: %s", userID)
		}

		return nil
	})
}

// GetRolesWithSubordinates ดึง Role ทั้งหมดพร้อมรายชื่อลูกน้อง
func (r *UserRepo) GetRolesWithSubordinates() ([]RoleWithMembersResponse, error) {
	// 1. ดึง Role ทั้งหมดมาก่อน
	var roles []Role
	if err := r.db.Order("role_id").Find(&roles).Error; err != nil {
		return nil, err
	}

	var response []RoleWithMembersResponse

	// 2. วนลูปแต่ละ Role
	for _, role := range roles {
		var subordinates []UserInfo

		// Join หาคนที่มี manager_role_id ตรงกับ Role นี้
		err := r.db.Table("user_info").
			Select("user_info.*").
			Joins("JOIN subordinate_manager_roles smr ON user_info.user_id = smr.subordinate_id").
			Where("smr.manager_role_id = ?", role.RoleID).
			Find(&subordinates).Error

		if err != nil {
			return nil, err
		}

		// 3. แปลงข้อมูลลูกน้อง (UserInfo) เป็น MemberResponse
		var members []MemberResponse = []MemberResponse{}

		for _, sub := range subordinates {
			members = append(members, MemberResponse{
				ID: sub.UserID,

				// 🚩 แก้ตรงนี้: เปลี่ยน n เล็ก เป็น N ใหญ่ ให้ตรงกับ Struct
				ThName: sub.FullNameThai,
				EnName: sub.FullNameEng,

				AvatarURL: sub.Picture,
			})
		}

		// 4. เพิ่มเข้า Response
		response = append(response, RoleWithMembersResponse{
			ID:        role.RoleID,
			RoleName:  role.RoleName,
			RoleColor: role.RoleColor,
			Type:      role.RoleType,
			Members:   members,
		})
	}

	return response, nil
}

// GetNonSubordinatesByRole ดึงรายชื่อพนักงานทั้งหมดที่ "ไม่ได้" เป็นลูกน้องของ Role ID ที่ระบุ
func (r *UserRepo) GetNonSubordinatesByRole(roleID string) ([]MemberResponse, error) {
	var members []MemberResponse

	// Query: เลือกคนจาก user_info ที่ user_id ไม่อยู่ในรายชื่อลูกน้องของ roleID นี้
	query := `
		SELECT 
			user_id as id, 
			fullname_thai as th_name, 
			fullname_eng as en_name, 
			picture as avatar_url
		FROM user_info
		WHERE user_id NOT IN (
			SELECT subordinate_id 
			FROM subordinate_manager_roles 
			WHERE manager_role_id = ?
		)
		ORDER BY fullname_thai ASC
	`

	err := r.db.Raw(query, roleID).Scan(&members).Error
	if err != nil {
		return nil, err
	}

	// ป้องกันค่า null ใน JSON ถ้าไม่มีข้อมูลให้ส่ง Array ว่างกลับไป
	if members == nil {
		members = []MemberResponse{}
	}

	return members, nil
}

func (r *UserRepo) UpdateRoleWithMembers(req UpdateRoleFullRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. อัปเดตข้อมูล Role (เหมือนเดิม)
		if err := tx.Table("role").Where("role_id = ?", req.ID).Updates(map[string]interface{}{
			"role_name":  req.Name,
			"role_type":  req.Type,
			"role_color": req.Color,
		}).Error; err != nil {
			return err
		}

		// 2. ล้างลูกน้องเก่าของ Role นี้ออกให้หมดก่อน (เหมือนเดิม)
		if err := tx.Table("subordinate_manager_roles").
			Where("manager_role_id = ?", req.ID).
			Delete(nil).Error; err != nil {
			return err
		}

		// 3. วนลูปจัดการสมาชิกใหม่
		if len(req.Members) > 0 {
			for _, member := range req.Members {

				// [Logic พิเศษ] ถ้า Role นี้เป็น Main ต้องไปลบ Main Role อื่นของลูกน้องทิ้งก่อน
				if req.Type == "main" {
					// ✅ แบบนี้ชัวร์ 100%
					subQuerySQL := "SELECT role_id FROM role WHERE role_type = 'main'"

					if err := tx.Table("subordinate_manager_roles").
						Where("subordinate_id = ?", member.ID).
						Where("manager_role_id IN (?)", gorm.Expr(subQuerySQL)). // ใช้ gorm.Expr หรือ string ตรงๆ
						Delete(nil).Error; err != nil {
						return err
					}
				}

				// 4. Insert สมาชิกใหม่ (ทำต่อเลยใน Loop เดียวกัน)
				newRelation := map[string]interface{}{
					"subordinate_id":  member.ID,
					"manager_role_id": req.ID,
				}
				if err := tx.Table("subordinate_manager_roles").Create(newRelation).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// DeleteRole ลบ Role และเคลียร์ความสัมพันธ์ลูกน้องทั้งหมด (Transaction)
func (r *UserRepo) DeleteRole(roleID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. ลบความสัมพันธ์ในตาราง subordinate_manager_roles ก่อน
		// (ถ้าไม่ลบก่อน จะติด Foreign Key หรือข้อมูลขยะค้าง)
		if err := tx.Table("subordinate_manager_roles").
			Where("manager_role_id = ?", roleID).
			Delete(nil).Error; err != nil {
			return err
		}

		// 2. ลบตัว Role จริงๆ ในตาราง role
		if err := tx.Table("role").
			Where("role_id = ?", roleID).
			Delete(nil).Error; err != nil {
			return err
		}

		return nil // Commit Transaction
	})
}

// GetUserRoles ดึง Role ทั้งหมดของ User ID นั้นๆ
func (r *UserRepo) GetUserRoles(userID string) ([]string, error) {
	var roles []string

	// 🚩 แก้ไขชื่อตารางจาก "roles" เป็น "role" (ไม่มี s) ให้ตรงกับ DB จริง
	err := r.db.Table("user_roles").
		Select("role.role_type").                                // เลือกจากตาราง role
		Joins("JOIN role ON role.role_id = user_roles.role_id"). // 🚩 แก้ JOIN role
		Where("user_roles.user_id = ?", userID).
		Pluck("role.role_type", &roles). // 🚩 แก้ Pluck role
		Error

	if err != nil {
		return nil, err
	}

	return roles, nil
}

// CreateRole สร้าง Role ใหม่และเพิ่มสมาชิก
func (r *UserRepo) CreateRole(req CreateRoleRequest) error {
	// เริ่ม Transaction
	tx := r.db.Begin()

	// 1. สร้าง Role
	role := Role{
		RoleID:    req.ID,
		RoleName:  req.Name,
		RoleColor: req.Color,
		RoleType:  req.Type,
	}

	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. เพิ่มสมาชิก (ถ้ามี)
	// สมมติว่า members คือ "ลูกน้องในสังกัด" (subordinate_manager_roles)
	// หรือถ้าหมายถึง "คนที่เป็น Role นี้" ให้เปลี่ยนไปใช้ตาราง user_roles แทนนะครับ
	if len(req.Members) > 0 {
		for _, member := range req.Members {
			// กรณี: เพิ่มลูกน้องในสังกัด (Subordinates)
			subordinate := SubordinateManagerRole{
				SubordinateID: member.ID,
				ManagerRoleID: req.ID,
			}
			if err := tx.Create(&subordinate).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Commit Transaction
	return tx.Commit().Error
}

// GetAllMembers ดึงรายชื่อพนักงานทั้งหมด (เฉพาะชื่อและรูป)
func (r *UserRepo) GetAllMembers() ([]MemberLite, error) {
	var members []MemberLite

	// Query เฉพาะ Column ที่จำเป็นจาก user_info
	// GORM จะ Map Column เข้า Struct ให้อัตโนมัติตาม Tag gorm:"column:..."
	err := r.db.Table("user_info").
		Select("user_id, fullname_thai, fullname_eng, picture").
		Scan(&members).Error

	return members, err
}

// 1. อัปเดต Path ลายเซ็น (ถ้าส่ง nil คือลบ)
func (r *UserRepo) UpdateSignaturePath(userID string, path *string) error {
	sql := `UPDATE users SET signature_path = $1 WHERE user_id = $2`
	return r.db.Exec(sql, path, userID).Error
}

// 2. ดึง Path ลายเซ็น
func (r *UserRepo) GetSignaturePath(userID string) (*string, error) {
	var path *string
	sql := `SELECT signature_path FROM users WHERE user_id = $1`
	err := r.db.Raw(sql, userID).Scan(&path).Error
	return path, err
}

// [NEW] GET /api/attendance/filter_range
func (r *UserRepo) GetAttendanceFilterRangeHistory(userID string) (map[string]interface{}, error) {
	var result struct {
		MinDate *time.Time `gorm:"column:min_date"`
		MaxDate *time.Time `gorm:"column:max_date"`
	}

	// ค้นหาวันที่น้อยที่สุด (เริ่ม) และมากที่สุด (ล่าสุด) จากตาราง attendance
	err := r.db.Table("attendance").
		Select("MIN(date) as min_date, MAX(date) as max_date").
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	// กำหนดค่า Default (เผื่อ User คนนี้ยังไม่เคยสแกนนิ้วเลย)
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)

	// ถ้ามีข้อมูลใน DB ให้เอาค่าจาก DB มาทับ
	if result.MinDate != nil {
		start = *result.MinDate
	}
	if result.MaxDate != nil {
		end = *result.MaxDate
	}

	// คืนค่ารูปแบบ "2025-04-01T00:00:00.000Z" ตาม Mock
	return map[string]interface{}{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	}, nil
}