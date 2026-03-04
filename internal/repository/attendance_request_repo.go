package repository

import (
	//"errors"
	"gorm.io/gorm"
	"time"
)

// สร้าง Struct สำหรับส่งข้อมูลไฟล์แนบ
type NewAttendanceAttachment struct {
	FilePath     string
	OriginalName string
	FileType     string
	FileSize     int64
}

// 1. ฟังก์ชันสร้างคำขอใหม่ (Create)
func (r *UserRepo) CreateAttendanceRequest(req *AttendanceRequest, attachments []NewAttendanceAttachment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1.1 บันทึกข้อมูลคำขอหลัก
		if err := tx.Table("attendance_requests").Create(req).Error; err != nil {
			return err
		}

		// 1.2 บันทึกไฟล์แนบ (ถ้ามี)
		for _, att := range attachments {
			if err := tx.Table("attendance_request_attachments").Create(map[string]interface{}{
				"attendance_request_id": req.ID,
				"file_path":             att.FilePath,
				"original_name":         att.OriginalName,
				"file_type":             att.FileType,
				"file_size":             att.FileSize,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// 2. ฟังก์ชันดึงรายการ Pending (รออนุมัติ)
func (r *UserRepo) GetPendingAttendanceRequests(userID string) ([]AttendanceRequest, error) {
	var requests []AttendanceRequest
	err := r.db.Table("attendance_requests").
		Where("user_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// 3. ฟังก์ชันดึงรายการ Recent (ประวัติที่อนุมัติ/ปฏิเสธแล้ว) พร้อม Filter วันที่
func (r *UserRepo) GetRecentAttendanceRequests(userID string, startDate, endDate *time.Time) ([]AttendanceRequest, error) {
	var requests []AttendanceRequest
	query := r.db.Table("attendance_requests").
		Where("user_id = ? AND status != ?", userID, "pending")

	if startDate != nil {
		query = query.Where("date_from >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("date_from <= ?", endDate)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

// 4. ฟังก์ชันหาช่วงวันที่ทั้งหมดที่มีข้อมูล (Filter Range)
func (r *UserRepo) GetAttendanceFilterRange(userID string) (time.Time, time.Time, error) {
	var result struct {
		MinDate time.Time
		MaxDate time.Time
	}
	
	err := r.db.Table("attendance_requests").
		Select("MIN(date_from) as min_date, MAX(date_from) as max_date").
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil || result.MinDate.IsZero() {
		// ถ้าไม่มีข้อมูลเลย ให้คืนค่า Default (เช่น ต้นปีนี้ ถึง สิ้นปีนี้)
		now := time.Now()
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), nil
	}
	return result.MinDate, result.MaxDate, nil
}