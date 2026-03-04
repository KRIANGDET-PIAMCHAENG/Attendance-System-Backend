package repository

import "time"

// 🌟 1. สร้าง Struct ใหม่สำหรับตาราง Approvals
type AttendanceApproval struct {
	ID                  uint      `gorm:"primaryKey"`
	AttendanceRequestID uint      `gorm:"column:attendance_request_id"`
	ApproverID          string    `gorm:"column:approver_id"`
	ApproveRole         string    `gorm:"column:approve_role"`
	Status              string    `gorm:"column:status"`
	Reason              string    `gorm:"column:reason"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

// 🌟 2. อัปเดต Model ตัวเดิม ให้รู้จัก Approval
type AttendanceRequest struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        string    `gorm:"column:user_id"`
	DateFrom      time.Time `gorm:"column:date_from"`
	DateTo        time.Time `gorm:"column:date_to"`
	StartTime     string    `gorm:"column:start_time"`
	EndTime       string    `gorm:"column:end_time"`
	Remark        string    `gorm:"column:remark"`
	SignaturePath string    `gorm:"column:signature_path"`
	Status        string    `gorm:"column:status;default:pending"`
	CreatedAt     time.Time `gorm:"column:created_at"`

	Attachments []AttendanceRequestAttachment `gorm:"foreignKey:AttendanceRequestID"`
	// เพิ่มบรรทัดนี้ลงไป (HasOne Relation)
	Approval    *AttendanceApproval           `gorm:"foreignKey:AttendanceRequestID"`
}

type AttendanceRequestAttachment struct {
    ID                  uint      `gorm:"primaryKey"`
    AttendanceRequestID uint      `gorm:"column:attendance_request_id"`
    FilePath            string    `gorm:"column:file_path"`
    OriginalName        string    `gorm:"column:original_name"`
    FileType            string    `gorm:"column:file_type"`
    FileSize            int64     `gorm:"column:file_size"`
    CreatedAt           time.Time `gorm:"column:created_at"`
}

// ==========================================
// Struct สำหรับตอบกลับ (Response) ให้ตรงกับ Mock Data
// ==========================================

// สำหรับเส้น /pending
type PendingAttendanceResponse struct {
    ID        string    `json:"id"`
    DateStart time.Time `json:"date-start"`
    DateEnd   time.Time `json:"date-end"`
}

// สำหรับเส้น /recent
type RecentAttendanceResponse struct {
    ID        string    `json:"id"`
    DateStart time.Time `json:"date-start"`
    DateEnd   time.Time `json:"date-end"`
    Approved  bool      `json:"approved"`
}