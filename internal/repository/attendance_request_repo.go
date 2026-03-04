package repository

import (
	"os"
	"errors"
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

// 5. ดึงรายละเอียดคำขอ (พร้อมไฟล์แนบ)
func (r *UserRepo) GetAttendanceDetail(userID string, reqID int) (*AttendanceRequest, error) {
	var request AttendanceRequest
	err := r.db.Preload("Attachments").
		Where("id = ? AND user_id = ?", reqID, userID).
		First(&request).Error
	return &request, err
}

// 6. ยกเลิกคำขอ (Soft Delete - เปลี่ยนสถานะเป็น canceled)
func (r *UserRepo) DeleteAttendanceRequest(userID string, reqID int) error {
	result := r.db.Table("attendance_requests").
		Where("id = ? AND user_id = ?", reqID, userID).
		Update("status", "canceled")

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("ไม่พบคำขอดังกล่าว หรือคุณไม่มีสิทธิ์")
	}
	return nil
}

// 7. ยื่นคำขอใหม่ (Resend)
func (r *UserRepo) ResendAttendanceRequest(userID string, reqID int, remark string, oldFiles []string, signaturePath *string, newFiles []NewAttendanceAttachment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 7.1 อัปเดตข้อมูลหลักให้กลับเป็น pending
		updates := map[string]interface{}{
			"status": "pending",
			"remark": remark,
		}
		if signaturePath != nil {
			updates["signature_path"] = *signaturePath
		}

		if err := tx.Table("attendance_requests").
			Where("id = ? AND user_id = ?", reqID, userID).
			Updates(updates).Error; err != nil {
			return err
		}

		// 7.2 จัดการลบไฟล์แนบเก่าที่ถูกกากบาททิ้ง
		var filesToDelete []string
		findQuery := tx.Table("attendance_request_attachments").Where("attendance_request_id = ?", reqID)
		if len(oldFiles) > 0 {
			findQuery = findQuery.Where("original_name NOT IN ?", oldFiles)
		}
		
		if err := findQuery.Pluck("file_path", &filesToDelete).Error; err != nil {
			return err
		}

		if len(filesToDelete) > 0 {
			if err := tx.Exec("DELETE FROM attendance_request_attachments WHERE file_path IN ?", filesToDelete).Error; err != nil {
				return err
			}
			for _, path := range filesToDelete {
				_ = os.Remove(path) // ลบไฟล์จริงออกจาก Harddisk
			}
		} else if len(oldFiles) == 0 {
			// ถ้าเก่าก็ลบหมด
			if err := tx.Exec("DELETE FROM attendance_request_attachments WHERE attendance_request_id = ?", reqID).Error; err != nil {
				return err
			}
		}

		// 7.3 เพิ่มไฟล์แนบใหม่ลง DB
		for _, att := range newFiles {
			if err := tx.Table("attendance_request_attachments").Create(map[string]interface{}{
				"attendance_request_id": reqID,
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