package repository

import (
	"errors"
	"fmt"
	"time"
	"my-app/internal/entity"
	"gorm.io/gorm"
	
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

type AttendanceRecord struct {
	Date     time.Time `gorm:"column:date"`
	CheckIn  *string   `gorm:"column:check_in"`  // 🚩 เปลี่ยนเป็น *string
	CheckOut *string   `gorm:"column:check_out"` // 🚩 เปลี่ยนเป็น *string
}

func (r *UserRepo) GetAttendanceHistory(userID string) ([]AttendanceRecord, error) {
	var records []AttendanceRecord
	
	sql := `
		SELECT date, check_in, check_out 
		FROM attendance 
		WHERE user_id = $1 
		ORDER BY date DESC
	`
	
	err := r.db.Raw(sql, userID).Scan(&records).Error
	return records, err
}

// [NEW] ฟังก์ชันสำหรับดึงข้อมูลการลงเวลาของ "วันนี้"
func (r *UserRepo) GetTodayAttendance(userID string, date time.Time) (*AttendanceRecord, error) {
	var record AttendanceRecord
	dateStr := date.Format("2006-01-02") // เอาแค่วันที่ (YYYY-MM-DD)

	// ค้นหาข้อมูลของ user คนนี้ เฉพาะวันที่กำหนด
	sql := `SELECT date, check_in, check_out FROM attendance WHERE user_id = $1 AND date = $2`
	result := r.db.Raw(sql, userID, dateStr).Scan(&record)

	if result.Error != nil {
		return nil, result.Error
	}

	// ถ้า RowsAffected == 0 แปลว่าวันนี้ยังไม่ได้กดอะไรเลย
	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &record, nil
}

// เช็คว่าเป็นวันหยุดหรือไม่
func (r *UserRepo) CheckHoliday(dateStr string) (*string, error) {
	var holiday CompanyHoliday
	
	// ค้นหาจากฐานข้อมูลตามวันที่ส่งมา
	err := r.db.Table("company_holidays").Where("holiday_date = ?", dateStr).First(&holiday).Error
	
	if err != nil {
		// ถ้าไม่เจอข้อมูล (ไม่ใช่วันหยุด) ให้คืนค่า nil (ซึ่งจะถูกแปลงเป็น null ใน JSON)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// ถ้า Error อื่นๆ (เช่น DB ล่ม)
		return nil, err
	}

	// ถ้าเจอ คืนค่าชื่อวันหยุดกลับไป
	return &holiday.Description, nil
}