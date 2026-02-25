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

// โครงสร้างสำหรับรับข้อมูลจาก DB
type LeaveStatusRecord struct {
	ID        int       `gorm:"column:id"`
	LeaveType string    `gorm:"column:leave_type"`
	DateStart time.Time `gorm:"column:date_from"`
	Status    string    `gorm:"column:status"`
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

// ตรวจสอบว่ามีใบลาที่ซ้อนทับกันอยู่หรือไม่
func (r *UserRepo) CheckOverlappingLeave(userID string, startDate string, endDate string) (bool, error) {
	// แปลง String เป็น time.Time ก่อน
	reqStart, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		return false, err
	}
	reqEnd, err := time.Parse(time.RFC3339, endDate)
	if err != nil {
		return false, err
	}

	var count int
	// ลอจิก overlap: (เก่าเริ่ม <= ใหม่จบ) AND (เก่าจบ >= ใหม่เริ่ม)
	// และต้องไม่นับใบลาที่ถูก rejected ไปแล้ว (เพราะถ้าถูกปฏิเสธ เขาควรมีสิทธิ์ยื่นใหม่ได้)
	sql := `
		SELECT COUNT(*) 
		FROM leave_requests 
		WHERE user_id = $1 
		  AND status != 'rejected' 
		  AND date_from <= $2 
		  AND date_to >= $3
	`
	
	// สังเกตการส่ง Parameter: $2 คือ reqEnd, $3 คือ reqStart
	if err := r.db.Raw(sql, userID, reqEnd, reqStart).Scan(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil // ถ้า count > 0 แปลว่ามีการซ้อนทับ
}

// 1. ดึงข้อมูลที่รออนุมัติ (Pending)
func (r *UserRepo) GetPendingLeaves(userID string) ([]LeaveStatusRecord, error) {
	var leaves []LeaveStatusRecord
	sql := `SELECT id, leave_type, date_from, status 
			FROM leave_requests 
			WHERE user_id = $1 AND status = 'pending' 
			ORDER BY date_from DESC`
	
	err := r.db.Raw(sql, userID).Scan(&leaves).Error
	return leaves, err
}

// 2. ดึงข้อมูลที่รู้ผลแล้ว (Recent) พร้อม Filter วันที่
func (r *UserRepo) GetRecentLeaves(userID string, startDate string, endDate string) ([]LeaveStatusRecord, error) {
	var leaves []LeaveStatusRecord
	
	query := `SELECT id, leave_type, date_from, status 
			  FROM leave_requests 
			  WHERE user_id = ? AND status != 'pending'`
	args := []interface{}{userID}

	if startDate != "" {
		query += ` AND date_from >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND date_from <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY date_from DESC`

	err := r.db.Raw(query, args...).Scan(&leaves).Error
	return leaves, err
}

// 3. หาช่วงวันที่ ที่เก่าที่สุดและใหม่ที่สุดของใบลายูสเซอร์คนนี้
func (r *UserRepo) GetLeaveFilterRange(userID string) (*time.Time, *time.Time, error) {
	var result struct {
		MinDate *time.Time `gorm:"column:min_date"`
		MaxDate *time.Time `gorm:"column:max_date"`
	}
	
	// หาค่าที่ต่ำสุดและสูงสุดจาก date_from ของยูสเซอร์คนนั้น
	sql := `SELECT MIN(date_from) as min_date, MAX(date_from) as max_date 
			FROM leave_requests 
			WHERE user_id = $1`
			
	err := r.db.Raw(sql, userID).Scan(&result).Error
	return result.MinDate, result.MaxDate, err
}