package entity

import "time"

type Attendance struct {
	UserID    string    `gorm:"primaryKey;column:user_id;type:varchar(50)"`
	Date      string    `gorm:"primaryKey;column:date;type:date"` // เก็บ "YYYY-MM-DD"
	CheckIn   string    `gorm:"column:check_in;type:time"`        // เก็บ "HH:mm:ss"
	CheckOut  *string   `gorm:"column:check_out;type:time"`       // Pointer เพราะเป็น NULL ได้
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Attendance) TableName() string {
	return "attendance"
}