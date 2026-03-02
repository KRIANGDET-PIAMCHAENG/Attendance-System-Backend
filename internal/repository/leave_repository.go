package repository

import (
	"errors"
	"time"
	"os"
	"gorm.io/gorm"
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

func (r *UserRepo) SaveLeaveAttachment(leaveID int, filePath string, fileName string, fileType string, fileSize int64) error {
    
    // 🌟 อัปเดตคำสั่ง SQL ให้ INSERT คอลัมน์ใหม่ลงไปด้วย
    sql := `
		INSERT INTO leave_attachments (
			leave_request_id, file_path, original_name, file_type, file_size
		) 
		VALUES ($1, $2, $3, $4, $5)
	`

    // 🌟 ส่งตัวแปร 5 ตัวเข้าไปให้ครบ
    if err := r.db.Exec(sql, leaveID, filePath, fileName, fileType, fileSize).Error; err != nil {
        return err
    }

    return nil
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

func (r *UserRepo) GetRecentLeaves(userID string, startDate string, endDate string) ([]LeaveStatusRecord, error) {
	var leaves []LeaveStatusRecord
	
	// Query: ดึงรายการที่ (ไม่ใช่ pending) OR (เป็น pending แต่เลยกำหนดวันลาแล้ว)
	query := `SELECT id, leave_type, date_from, status 
			  FROM leave_requests 
			  WHERE user_id = ? 
			    AND (status != 'pending' OR (status = 'pending' AND date_from < CURRENT_TIMESTAMP))`
	
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


// 1. สร้าง Struct มารับข้อมูล
type LeaveDetailRecord struct {
	ID              int
	LeaveType       string
	DateFrom        time.Time
	DateTo          time.Time
	FromDateMorning bool
	ToDateMorning   bool
	Remark          string
	CreatedAt       time.Time
	Status          string
	// ฟิลด์จากตาราง leave_approvals (ใช้ Pointer เผื่อยังไม่มีคนมาอนุมัติ จะได้เป็น nil ได้)
	Approver        *string
	ApproveRole     *string
	ApproveReason   *string
	ApproveDate     *time.Time
}

type LeaveAttachmentRecord struct {
    OriginalName string `gorm:"column:original_name"`
    FilePath     string `gorm:"column:file_path"`
    FileType     string `gorm:"column:file_type"` // เพิ่ม tag
    FileSize     int64  `gorm:"column:file_size"` // เพิ่ม tag
}

// 2. ฟังก์ชัน GetLeaveDetail (ใช้ LEFT JOIN)
func (r *UserRepo) GetLeaveDetail(userID string, leaveID int) (*LeaveDetailRecord, []LeaveAttachmentRecord, error) {
	var detail LeaveDetailRecord
	
	// ดึงข้อมูลใบลาหลัก พร้อมข้อมูลผู้อนุมัติล่าสุด (ถ้ามี)
	err := r.db.Table("leave_requests lr").
		Select(`
			lr.id, lr.leave_type, lr.date_from, lr.date_to, 
			lr.from_date_morning, lr.to_date_morning, lr.remark, lr.created_at, lr.status,
			la.approver_name as approver, 
			la.approve_role, 
			la.reason as approve_reason, 
			la.created_at as approve_date
		`).
		Joins("LEFT JOIN leave_approvals la ON la.leave_request_id = lr.id").
		Where("lr.id = ? AND lr.user_id = ?", leaveID, userID).
		Order("la.created_at DESC"). // ถ้ามีการอนุมัติหลายรอบ เอาล่าสุดมาแสดง
		First(&detail).Error

	if err != nil {
		return nil, nil, err
	}

	// ดึงข้อมูลไฟล์แนบ (หมายเหตุ: ถ้าตาราง leave_attachments ของคุณไม่มีฟิลด์ file_size หรือ file_type ให้ลบออกจาก Select ตรงนี้ด้วยนะครับ)
	var attachments []LeaveAttachmentRecord
	r.db.Table("leave_attachments").
        Select("original_name, file_path, file_type, file_size"). // ✅ เพิ่ม 2 คอลัมน์นี้เข้าไป
        Where("leave_request_id = ?", leaveID).
        Find(&attachments)

	return &detail, attachments, nil
}

// สร้าง Struct ไว้รับค่าที่ Query ออกมา
type LeaveBalanceInfo struct {
	DaysUsed    float64 `gorm:"column:days_used"`
	DaysAllowed float64 `gorm:"column:days_allowed"`
}

func (r *UserRepo) GetLeaveBalanceInfo(userID string, leaveTypeStr string) (float64, float64, error) {
	var typeID int

	// 1. หา ID จากตาราง leave_types โดยใช้ชื่อภาษาอังกฤษ (ป้องกัน case sensitive)
	if err := r.db.Table("leave_types").Select("id").Where("LOWER(name_en) = LOWER(?)", leaveTypeStr).Scan(&typeID).Error; err != nil || typeID == 0 {
		return 0, 0, errors.New("ไม่พบประเภทการลานี้ในระบบ")
	}

	// 2. คำนวณหา "ปีงบประมาณปัจจุบัน" แบบ Real-time
	currentYear, err := GetBudgetYear(r.db, time.Now())
	if err != nil {
		return 0, 0, errors.New("ไม่สามารถคำนวณปีงบประมาณได้")
	}

	// 3. ดึงโควต้าวันลาของ User คนนี้ เฉพาะปีงบประมาณปัจจุบันเท่านั้น!
	var balance LeaveBalanceInfo
	err = r.db.Table("leave_balances").
		Select("days_used, days_allowed").
		Where("user_id = ? AND leave_type_id = ? AND year = ?", userID, typeID, currentYear).
		First(&balance).Error

	if err != nil {
		// ถ้าหาไม่เจอ (เช่น เพิ่งขึ้นปีงบประมาณใหม่ แต่ระบบยังไม่ได้รันโควต้าให้)
		// เราสามารถ Return 0, 0 ไปก่อนเพื่อไม่ให้แอปฝั่ง Frontend พังครับ
		return 0, 0, nil 
	}

	return balance.DaysUsed, balance.DaysAllowed, nil
}

// สร้าง Struct เพื่อรับไฟล์ใหม่ที่จะ Insert
type NewAttachment struct {
	FilePath     string
	OriginalName string
	FileType     string
	FileSize     int64
}

func (r *UserRepo) ResendLeaveRequest(userID string, leaveID int, remark string, oldFiles []string, signaturePath *string, newFiles []NewAttachment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. อัปเดตข้อมูลใบลาหลัก (รีเซ็ตสถานะเป็น pending)
		updates := map[string]interface{}{
			"status": "pending",
			"remark": remark,
		}
		if signaturePath != nil {
			updates["signature_path"] = *signaturePath
		}

		if err := tx.Table("leave_requests").
			Where("id = ? AND user_id = ?", leaveID, userID).
			Updates(updates).Error; err != nil {
			return err
		}

		// 2. จัดการไฟล์เก่า (หาไฟล์ที่ต้อง "ลบทิ้ง")
		var filesToDelete []string
		query := tx.Table("leave_attachments").Where("leave_request_id = ?", leaveID)
		
		// ถ้ามีไฟล์เก่าที่ต้องการเก็บไว้ ให้หาไฟล์ที่ชื่อ "ไม่อยู่" ในลิสต์นั้น
		if len(oldFiles) > 0 {
			query = query.Where("original_name NOT IN ?", oldFiles)
		}

		// ดึง path ของไฟล์ที่กำลังจะโดนลบออกมาเก็บไว้ก่อน เพื่อไปลบออกจาก Harddisk
		if err := query.Pluck("file_path", &filesToDelete).Error; err != nil {
			return err
		}

		// ลบข้อมูลไฟล์ออกจาก Database
		if err := query.Delete(nil).Error; err != nil {
			return err
		}

		// 🗑️ ลบไฟล์จริงออกจาก Server (Best Effort: ลบได้ก็ลบ ลบไม่ได้ไม่เป็นไร ไม่ให้กระทบ DB)
		for _, path := range filesToDelete {
			_ = os.Remove(path)
		}

		// 3. เพิ่มไฟล์ใหม่ (ถ้ามี) ลง Database
		for _, att := range newFiles {
			if err := tx.Table("leave_attachments").Create(map[string]interface{}{
				"leave_request_id": leaveID,
				"file_path":        att.FilePath,
				"original_name":    att.OriginalName,
				"file_type":        att.FileType,
				"file_size":        att.FileSize,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}