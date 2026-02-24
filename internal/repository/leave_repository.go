package repository

import (
	"errors"
	"time"
)

type CreateLeaveRequest struct {
	LeaveType       string `form:"leave-type" binding:"required"`
	DateFrom        string `form:"date-from" binding:"required"` 
	DateTo          string `form:"date-to" binding:"required"`   
	FromDateMorning bool   `form:"from-date-morning"`
	ToDateMorning   bool   `form:"to-date-morning"`
	Remark          string `form:"remark"`
}

// เพิ่ม signaturePath *string เป็น Parameter ตัวที่ 3
func (r *UserRepo) SaveLeaveRequest(userID string, req CreateLeaveRequest, signaturePath *string) (int, error) {
	var id int

	dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
	if err != nil {
		return 0, errors.New("invalid date-from format")
	}

	dateTo, err := time.Parse(time.RFC3339, req.DateTo)
	if err != nil {
		return 0, errors.New("invalid date-to format")
	}

	// 🌟 เพิ่มคอลัมน์ signature_path เข้าไปใน SQL
	sql := `
		INSERT INTO leave_requests (
			user_id, leave_type, date_from, date_to, 
			from_date_morning, to_date_morning, remark, signature_path
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		RETURNING id
	`

	// 🌟 ส่งค่า signaturePath เข้าไปเป็น parameter ตัวที่ 8
	if err := r.db.Raw(sql,
		userID,
		req.LeaveType,
		dateFrom,
		dateTo,
		req.FromDateMorning,
		req.ToDateMorning,
		req.Remark,
		signaturePath,
	).Scan(&id).Error; err != nil {
		return 0, err
	}

	return id, nil
}

// SaveLeaveAttachment เหมือนเดิม
func (r *UserRepo) SaveLeaveAttachment(leaveID int, path string, originalName string) error {
	sql := `
		INSERT INTO leave_attachments (leave_request_id, file_path, original_name) 
		VALUES ($1, $2, $3)
	`
	return r.db.Exec(sql, leaveID, path, originalName).Error
}