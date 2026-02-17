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

// TimePair เก็บเวลา (ชั่วโมง:นาที)
type TimePair struct {
	Hour   int `json:"hour" binding:"min=0,max=23"`   // บังคับ 0-23
	Minute int `json:"minute" binding:"min=0,max=59"` // บังคับ 0-59
}

// ConfigAttendanceTime โครงสร้างข้อมูลตั้งค่าเวลาเข้างาน
type ConfigAttendanceTime struct {
	AutoCheckout      bool     `json:"auto-checkout"`
	CutoffTime        TimePair `json:"cutoff-time" binding:"required"`
	CheckInTime       TimePair `json:"check-in-time" binding:"required"`
	CheckOutTime      TimePair `json:"check-out-time" binding:"required"`
	CheckInLeaveTime  TimePair `json:"check-in-leave-time" binding:"required"`
	CheckOutLeaveTime TimePair `json:"check-out-leave-time" binding:"required"`
}

type ConfigAttendanceRequest struct {
	RequestNeedSignature  bool `json:"request-need-signature"`
	ApproveNeedSignature  bool `json:"approve-need-signature"`
	SpecifyApprovalReason bool `json:"specify-approval-reason"`
	SpecifyRemark         bool `json:"specify-remark"`
	RequiredRemark        bool `json:"required-remark"`
	EvidenceFile          bool `json:"evidence-file"`
	RequiredEvidenceFile  bool `json:"required-evidence-file"`
}

// LeaveSettings รายละเอียดการตั้งค่าของแต่ละประเภทการลา
type LeaveSettings struct {
	RequestNeedSignature bool `json:"request-need-signature"`
	ApproveNeedSignature bool `json:"approve-need-signature"`
	AllowRetroactive     bool `json:"allow-retroactive"`
	SpecifyRemark        bool `json:"specify-remark"`
	RequiredRemark       bool `json:"required-remark"`
	EvidenceFile         bool `json:"evidence-file"`
	RequiredEvidenceFile bool `json:"required-evidence-file"`
}

// ConfigLeave คือ Map ที่เก็บ LeaveSettings แยกตามชื่อประเภทการลา
// เช่น data["sick"] = LeaveSettings{...}
type ConfigLeave map[string]LeaveSettings