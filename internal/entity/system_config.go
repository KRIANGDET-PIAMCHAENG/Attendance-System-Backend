package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// --- Table Schema ---

// SystemConfig ตารางเก็บค่า Config ทั้งหมด (Key-Value)
type SystemConfig struct {
	ConfigKey   string    `gorm:"primaryKey;column:config_key;type:varchar(50)"`
	ConfigValue JSONB     `gorm:"column:config_value;type:jsonb"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// กำหนดชื่อตาราง
func (SystemConfig) TableName() string {
	return "system_configs"
}

// --- Helper for JSONB ---
// เพื่อให้ GORM เข้าใจ JSONB ของ Postgres
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// --- Payload Structs (Data Format) ---

// ConfigBudgetYear โครงสร้างข้อมูลสำหรับ Budget Year
type ConfigBudgetYear struct {
	Day   int `json:"day" binding:"required"`
	Month int `json:"month" binding:"required"`
}