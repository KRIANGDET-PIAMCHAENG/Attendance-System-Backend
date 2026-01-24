package entity


type UserInfo struct{
	
	UserID string `gorm:"primaryKey;column:user_id" json:"user_id"`
	EmployeeID string `gorm:"column:employee_id" json:"employee_id"`
	Email string `gorm:"column:email" json:"email"`
	FullNameThai string `gorm:"column:fullname_thai" json:"fullname_thai"`
	FullNameEng string `gorm:"column:fullname_eng" json:"fullname_eng"`
	Gender string `gorm:"column:gender" json:"gender"`
	Nationality string `gorm:"column:nationality" json:"nationality"`
	Phone string `gorm:"column:phone" json:"phone"`
	RoleInit string `gorm:"column:role_init" json:"role_init"`
	
}

// Role: ตารางสิทธิ์/ตำแหน่ง
type Role struct {
	RoleID    string `gorm:"primaryKey;column:role_id" json:"role_id"`
	RoleName  string `gorm:"column:role_name" json:"role_name"`
	RoleColor string `gorm:"column:role_color" json:"role_color"`
	RoleType  string `gorm:"column:role_type" json:"role_type"`
}

// User: ตารางหลักสำหรับโครงสร้างสายงาน
type User struct {
	UserID        string `gorm:"primaryKey;column:user_id" json:"user_id"`
	EmployeeID    string `gorm:"column:employee_id" json:"employee_id"`
	ManagerRoleID string `gorm:"column:manager_role_id" json:"manager_role_id"`
}

// UserRole: ตารางกลางสำหรับเชื่อม User กับ Role (Many-to-Many relationship)
type UserRole struct {
	ID        int       `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	UserID    string    `gorm:"column:user_id" json:"user_id"`
	RoleID    string    `gorm:"column:role_id" json:"role_id"`
}