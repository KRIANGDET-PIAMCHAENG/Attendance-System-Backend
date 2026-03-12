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

// 5. ดึงรายละเอียดคำขอ (พร้อมไฟล์แนบ + ข้อมูลผู้อนุมัติ)
// 🌟 [แก้ตรงนี้] เพิ่ม return string อีก 1 ตัว สำหรับส่งค่า expectedRole กลับไป
func (r *UserRepo) GetAttendanceDetail(userID string, reqID int) (*AttendanceRequest, string, string, error) {
	var request AttendanceRequest
	// ใช้ Preload ดึงทั้ง Attachments และ Approval มาพร้อมกัน
	err := r.db.Preload("Attachments").Preload("Approval").
		Where("id = ? AND user_id = ?", reqID, userID).
		First(&request).Error

	if err != nil {
		return nil, "", "", err
	}

	var approverName string
	var expectedRole string

	// ถ้ามีคนอนุมัติแล้ว (ประวัติถูกสร้างแล้ว)
	if request.Approval != nil && request.Approval.ApproverID != "" {
		// ⚠️ หมายเหตุ: ปรับโค้ดใน Select ให้ตรงกับชื่อคอลัมน์ในตาราง user_info ของคุณ
		r.db.Table("user_info").
			Select("fullname_thai"). // หรือ "first_name || ' ' || last_name" ตามที่ลูกพี่ใช้
			Where("user_id = ?", request.Approval.ApproverID).
			Scan(&approverName)

		expectedRole = request.Approval.ApproveRole
	} else {
		// 🌟 [NEW LOGIC] ถ้ายังไม่มีประวัติการอนุมัติ ให้ไปหาตำแหน่งหัวหน้าล่วงหน้า
		r.db.Table("subordinate_manager_roles smr").
			Select("r.role_name").
			Joins("JOIN role r ON smr.manager_role_id = r.role_id").
			Where("smr.subordinate_id = ? AND r.role_type = ?", userID, "main").
			Limit(1).
			Scan(&expectedRole)
	}

	// คืนค่ากลับไป 4 อย่าง: ข้อมูลคำขอ, ชื่อคนอนุมัติ, ตำแหน่งผู้อนุมัติ(ล่วงหน้า), และ error
	return &request, approverName, expectedRole, nil
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

// สร้าง Struct จำลองให้ตรงกับที่ Flutter ต้องการ (อันนี้ถูกแล้ว ไว้ข้างบนฟังก์ชัน)
type LeaveCalendarData struct {
    IsApproved bool   `json:"is_approved"`
    LeaveType  string `json:"leave_type"` // FULL_DAY, MORNING, AFTERNOON
    LeaveName  string `json:"leave_name"`
}

// 🌟 [แก้ตรงนี้] เปลี่ยนจาก (r *AttendanceReqRepo) เป็น (r *UserRepo)
func (r *UserRepo) GetLeaveCalendarData(userID string, targetYear int) (map[string]LeaveCalendarData, error) {
    // 1. กำหนดหัว-ท้าย ของปีเป้าหมาย
    startDate := time.Date(targetYear, 1, 1, 0, 0, 0, 0, time.Local)
    endDate := time.Date(targetYear, 12, 31, 23, 59, 59, 0, time.Local)

    var requests []struct {
        LeaveType       string    `gorm:"column:leave_type"`
        DateFrom        time.Time `gorm:"column:date_from"`
        DateTo          time.Time `gorm:"column:date_to"`
        FromDateMorning bool      `gorm:"column:from_date_morning"`
        ToDateMorning   bool      `gorm:"column:to_date_morning"`
        Status          string    `gorm:"column:status"`
    }

    // 2. ดึงใบลาเฉพาะปีนั้นๆ มา (เอาทั้งที่อนุมัติแล้ว และรออนุมัติ)
    err := r.db.Table("leave_requests").
        Select("leave_type, date_from, date_to, from_date_morning, to_date_morning, status").
        Where("user_id = ? AND status IN ('approved', 'pending') AND date_from <= ? AND date_to >= ?", userID, endDate, startDate).
        Scan(&requests).Error

    if err != nil {
        return nil, err
    }

    // 3. สร้าง Map เก็บข้อมูลโดยมี Key เป็น "YYYY-MM-DD"
    result := make(map[string]LeaveCalendarData)

    for _, req := range requests {
        isApproved := req.Status == "approved"
        leaveName := req.LeaveType

        // 🌟 ลูปตั้งแต่วันเริ่มลา จนถึงวันสิ้นสุดลา
        for d := req.DateFrom; !d.After(req.DateTo); d = d.AddDate(0, 0, 1) {
            // ข้ามถ้าข้ามปี
            if d.Year() != targetYear {
                continue
            }

            dateStr := d.Format("2006-01-02") // format เป็น Key
            leaveTypeStr := "FULL_DAY"        // Default ลาเต็มวัน

            isStartDay := d.Format("2006-01-02") == req.DateFrom.Format("2006-01-02")
            isEndDay := d.Format("2006-01-02") == req.DateTo.Format("2006-01-02")

            // 🎯 Logic แยก ลาเต็มวัน / ครึ่งเช้า / ครึ่งบ่าย
            if isStartDay && isEndDay {
                if req.FromDateMorning && req.ToDateMorning {
                    leaveTypeStr = "MORNING"
                } else if !req.FromDateMorning && !req.ToDateMorning {
                    leaveTypeStr = "AFTERNOON"
                }
            } else {
                if isStartDay && !req.FromDateMorning {
                    leaveTypeStr = "AFTERNOON"
                } else if isEndDay && req.ToDateMorning {
                    leaveTypeStr = "MORNING"
                }
            }

            result[dateStr] = LeaveCalendarData{
                IsApproved: isApproved,
                LeaveType:  leaveTypeStr,
                LeaveName:  leaveName,
            }
        }
    }

    return result, nil
}