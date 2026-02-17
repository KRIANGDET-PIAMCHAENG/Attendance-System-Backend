package repository

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
	ID          string `json:"id"`
	NameTH      string `json:"name-th"`
	NameEN      string `json:"name-en"`
	AvatarURL   string `json:"avatar-url"`
	EmployeeID  string `json:"employee-id"`
	Gender      string `json:"gender"`
	Nationality string `json:"nationality"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	InitialRole string `json:"initial-role"`

	// 🚩 เติม gorm:"-" ต่อท้ายแบบนี้ครับ
	Roles []RoleAPI `json:"roles" gorm:"-"`
}

type RoleMemberAPI struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Color  string `json:"color"`
	Member int    `json:"member"` // ตัวนี้จะเก็บค่าจากการ COUNT
}

type LeaveQuotaResult struct {
	TypeKey     string  `gorm:"column:name_en"`      // เช่น sick, personal
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

type MemberResponse struct {
	ID        string `json:"id"`
	ThName    string `json:"thName"`
	EnName    string `json:"enName"`
	AvatarURL string `json:"avatarUrl"`
}

// RoleWithMembersResponse โครงสร้างข้อมูล Role พร้อมลูกน้อง
type RoleWithMembersResponse struct {
	ID        string           `json:"id"`
	RoleName  string           `json:"roleName"`
	RoleColor string           `json:"roleColor"`
	Type      string           `json:"type"`
	Members   []MemberResponse `json:"members"`
}

// Role เป็น Struct ที่ map กับตาราง "role" ใน Database
type Role struct {
	RoleID    string `gorm:"primaryKey;column:role_id"`
	RoleName  string `gorm:"column:role_name"`
	RoleColor string `gorm:"column:role_color"`
	RoleType  string `gorm:"column:role_type"`
}

type MemberIDReq struct {
	ID string `json:"id"`
}

type UpdateRoleFullRequest struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Color   string        `json:"color"`
	Members []MemberIDReq `json:"members"` // รับเป็น Array Object: [{"id": "..."}]
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

// CreateRoleRequest รับข้อมูลสร้าง Role จาก Frontend
type CreateRoleRequest struct {
	ID      string       `json:"id" binding:"required"`
	Type    string       `json:"type" binding:"required"`
	Color   string       `json:"color" binding:"required"`
	Name    string       `json:"name" binding:"required"`
	Members []MemberItem `json:"members"` // รับ Array ของ User ID
}

// MemberItem โครงสร้างย่อยสำหรับสมาชิกใน Role
type MemberItem struct {
	ID string `json:"id"`
}

// ... (ต่อท้ายไฟล์ user.go)

type SubordinateManagerRole struct {
	SubordinateID string    `gorm:"primaryKey;column:subordinate_id;type:varchar(50)"`
	ManagerRoleID string    `gorm:"primaryKey;column:manager_role_id;type:varchar(50)"`
    // CreatedAt     time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"` // ถ้า DB มี field นี้ก็ใส่
}

// กำหนดชื่อตารางให้ตรงกับ DB
func (SubordinateManagerRole) TableName() string {
	return "subordinate_manager_roles"
}