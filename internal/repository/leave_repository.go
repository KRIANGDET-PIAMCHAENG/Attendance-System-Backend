package repository

import (
	"errors"
	"time"
)

// CreateLeaveRequest: Struct นี้ใช้รับข้อมูลจาก Frontend (ผ่าน Handler)
// หมายเหตุ: ถ้าคุณเอา Struct นี้ไปไว้ใน user_types.go แล้ว สามารถลบส่วนนี้ทิ้งได้ครับ
type CreateLeaveRequest struct {
	LeaveType       string `form:"leave-type" binding:"required"`
	DateFrom        string `form:"date-from" binding:"required"` // ISO8601 String
	DateTo          string `form:"date-to" binding:"required"`   // ISO8601 String
	FromDateMorning bool   `form:"from-date-morning"`
	ToDateMorning   bool   `form:"to-date-morning"`
	Remark          string `form:"remark"`
}

// 1. บันทึกข้อมูลการลาลง Database
// เปลี่ยนชื่อเป็น SaveLeaveRequest ให้ตรงกับ Handler
// เพิ่ม parameter userID เพราะต้องรู้ว่าใครเป็นคนลา
func (r *UserRepo) SaveLeaveRequest(userID string, req CreateLeaveRequest) (int, error) {
	var id int

	// 1. แปลง String (ISO8601) เป็น Time Object เพื่อความชัวร์
	// layout "2006-01-02T15:04:05Z" หรือ time.RFC3339
	dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
	if err != nil {
		return 0, errors.New("invalid date-from format")
	}

	dateTo, err := time.Parse(time.RFC3339, req.DateTo)
	if err != nil {
		return 0, errors.New("invalid date-to format")
	}

	// 2. SQL Query (ใช้ Raw SQL เพราะต้องการ RETURNING id)
	sql := `
		INSERT INTO leave_requests (
			user_id, leave_type, date_from, date_to, 
			from_date_morning, to_date_morning, remark
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING id
	`

	// 3. Execute และ Scan ID ที่ได้กลับมา
	// ใช้ r.db.Raw สำหรับ Query ที่มีการ return ค่ากลับมา
	if err := r.db.Raw(sql, 
		userID, 
		req.LeaveType, 
		dateFrom, 
		dateTo, 
		req.FromDateMorning, 
		req.ToDateMorning, 
		req.Remark,
	).Scan(&id).Error; err != nil {
		return 0, err
	}

	return id, nil
}

// 2. บันทึกไฟล์แนบ (Attachments)
func (r *UserRepo) SaveLeaveAttachment(leaveID int, path string, originalName string) error {
	sql := `
		INSERT INTO leave_attachments (leave_request_id, file_path, original_name) 
		VALUES ($1, $2, $3)
	`
	// ใช้ r.db.Exec สำหรับ Query ที่ไม่ต้อง return ค่าอะไรกลับมา (นอกจาก error)
	return r.db.Exec(sql, leaveID, path, originalName).Error
}