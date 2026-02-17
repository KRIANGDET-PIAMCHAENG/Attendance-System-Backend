package repository

import (
	"errors"
	"fmt"
	"time"
	"my-app/internal/entity"
	//"gorm.io/gorm"
)

// Struct รับ JSON ตัวใหม่
type RecordAttendanceRequest struct {
	Timestamp string `json:"timestamp" binding:"required"` // ISO8601 ex: "2026-02-18T10:30:00.000Z"
	Type      string `json:"type" binding:"required"`      // "CHECK_IN" or "CHECK_OUT"
}

func (r *UserRepo) RecordAttendance(userID string, req RecordAttendanceRequest) error {
	
	// 1. Parse ISO String เป็น Time Object
	parsedTime, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		return errors.New("Invalid timestamp format (ISO 8601 required)")
	}

	// 2. แปลงเป็นเวลาไทย (Asia/Bangkok) 
	// สำคัญมาก! ไม่งั้น 06:00 เช้า (ไทย) จะกลายเป็น 23:00 เมื่อวาน (UTC)
	loc, _ := time.LoadLocation("Asia/Bangkok")
	localTime := parsedTime.In(loc)

	// 3. แยกชิ้นส่วน Date และ Time
	dateOnly := localTime.Format("2006-01-02") // "2026-02-18"
	timeOnly := localTime.Format("15:04:05")   // "10:30:00"

	// 4. บันทึกลง DB
	if req.Type == "CHECK_IN" {
		attendance := entity.Attendance{
			UserID:  userID,
			Date:    dateOnly,
			CheckIn: timeOnly,
			// CheckOut ปล่อย NULL
		}
		// Create
		return r.db.Create(&attendance).Error

	} else if req.Type == "CHECK_OUT" {
		// Update
		result := r.db.Model(&entity.Attendance{}).
			Where("user_id = ? AND date = ?", userID, dateOnly).
			Updates(map[string]interface{}{
				"check_out": timeOnly,
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("ไม่พบข้อมูล Check-in ของวันที่ %s", dateOnly)
		}
	}

	return nil
}