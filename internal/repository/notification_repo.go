package repository

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// โครงสร้างตาราง Notification
type Notification struct {
	ID            string    `gorm:"column:id;primaryKey"`
	UserID        string    `gorm:"column:user_id"`
	Title         string    `gorm:"column:title"`
	Message       string    `gorm:"column:message"`
	IsRead        bool      `gorm:"column:is_read"`
	Type          string    `gorm:"column:type"`
	Status        string    `gorm:"column:status"`
	RequestNumber string    `gorm:"column:request_number"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// 1. GET /notifications
func (r *NotificationRepo) GetNotifications(userID string) ([]map[string]interface{}, error) {
	var notifs []Notification
	r.db.Table("notifications").Where("user_id = ?", userID).Order("created_at DESC").Scan(&notifs)

	var results []map[string]interface{}
	for _, n := range notifs {
		results = append(results, map[string]interface{}{
			"id":            n.ID,
			"title":         n.Title,
			"message":       n.Message,
			"isRead":        n.IsRead,
			"type":          n.Type,
			"status":        n.Status,
			"requestNumber": n.RequestNumber,
			"createdAt":     n.CreatedAt.Format(time.RFC3339),
		})
	}
	
	if len(results) == 0 {
		return []map[string]interface{}{}, nil
	}
	return results, nil
}

// 2. PATCH /notifications/{id}/read
func (r *NotificationRepo) MarkAsRead(userID, notifID string) error {
	res := r.db.Table("notifications").Where("id = ? AND user_id = ?", notifID, userID).Update("is_read", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("not found")
	}
	return nil
}

// 3. PATCH /notifications/read-all
func (r *NotificationRepo) MarkAllAsRead(userID string) error {
	return r.db.Table("notifications").Where("user_id = ? AND is_read = false", userID).Update("is_read", true).Error
}

// 4. GET /notifications/unread-count
func (r *NotificationRepo) GetUnreadCount(userID string) (int64, error) {
	var count int64
	err := r.db.Table("notifications").Where("user_id = ? AND is_read = false", userID).Count(&count).Error
	return count, err
}
// ==========================================
// 🌟 ฟังก์ชันเสริม: แปลงวันที่เป็น Format ภาษาไทย
// ==========================================
func formatThaiDateRange(start, end time.Time) string {
	thaiMonths := []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}
	
	startStr := fmt.Sprintf("%d %s %d", start.Day(), thaiMonths[start.Month()-1], start.Year()+543)
	endStr := fmt.Sprintf("%d %s %d", end.Day(), thaiMonths[end.Month()-1], end.Year()+543)

	if startStr == endStr {
		return fmt.Sprintf("วันที่ %s", startStr)
	}
	return fmt.Sprintf("วันที่ %s ถึงวันที่ %s", startStr, endStr)
}

// ==========================================
// 5. POST /api/notifications/send-request (ส่งหา หัวหน้า)
// ==========================================
func (r *NotificationRepo) CreateRequestNotification(reqType, requestNumber string) error {
	prefix := "LEV"
	tableName := "leave_requests"
	if strings.HasPrefix(requestNumber, "REQ") {
		prefix = "REQ"
		tableName = "attendance_requests"
	}

	idStr := strings.TrimLeft(strings.TrimPrefix(requestNumber, prefix), "0")
	reqID, _ := strconv.Atoi(idStr)

	// ดึงข้อมูลคำขอ + ชื่อคนส่ง
	var req struct {
		UserID       string    `gorm:"column:user_id"`
		DateFrom     time.Time `gorm:"column:date_from"`
		DateTo       time.Time `gorm:"column:date_to"`
		LeaveType    string    `gorm:"column:leave_type"`
		RequesterName string   `gorm:"column:fullname_thai"`
	}

	query := r.db.Table(tableName+" req").
		Select("req.user_id, req.date_from, req.date_to, ui.fullname_thai").
		Joins("JOIN user_info ui ON req.user_id = ui.user_id").
		Where("req.id = ?", reqID)

	if prefix == "LEV" {
		query.Select("req.user_id, req.date_from, req.date_to, req.leave_type, ui.fullname_thai")
	}
	
	if err := query.Scan(&req).Error; err != nil || req.UserID == "" {
		return errors.New("request not found")
	}

	// สร้าง Title และ Message ตาม Format
	var title, message string
	dateStr := formatThaiDateRange(req.DateFrom, req.DateTo)

	if reqType == "APPROVER_LEAVE" {
		title = "การอนุมัติคำขอลางาน"
		
		leaveName := "ลางาน"
		r.db.Table("leave_types").Where("name_en = ?", req.LeaveType).Select("name_th").Scan(&leaveName)
		
		// 🌟 [NEW] เช็คว่าถ้าไม่มีคำว่า "ลา" นำหน้า ให้เติมเข้าไป (พักผ่อน -> ลาพักผ่อน)
		if !strings.HasPrefix(leaveName, "ลา") {
			leaveName = "ลา" + leaveName
		}
		
		// 🌟 [NEW] เพิ่มคำว่า "กำลังรอการตรวจสอบจากคุณ"
		message = fmt.Sprintf("คำขอ%s%s โดย %s กำลังรอการตรวจสอบจากคุณ", leaveName, dateStr, req.RequesterName)
	} else {
		title = "การอนุมัติเวลาเข้า-ออกงาน"
		// 🌟 [NEW] เพิ่มคำว่า "กำลังรอการตรวจสอบจากคุณ"
		message = fmt.Sprintf("คำขออนุมัติเวลาเข้า-ออกงาน%s โดย %s กำลังรอการตรวจสอบจากคุณ", dateStr, req.RequesterName)
	}

	// หาหัวหน้า (role_type = 'main')
	var managerIDs []string
	r.db.Table("subordinate_manager_roles smr").
		Select("DISTINCT ur.user_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("smr.subordinate_id = ? AND r.role_type = ?", req.UserID, "main").
		Pluck("user_id", &managerIDs)

	for _, managerID := range managerIDs {
		notif := Notification{
			ID:            "notif_" + uuid.New().String()[:8],
			UserID:        managerID,
			Title:         title,
			Message:       message,
			IsRead:        false,
			Type:          reqType,
			Status:        "PENDING",
			RequestNumber: requestNumber,
			CreatedAt:     time.Now(),
		}
		if err := r.db.Table("notifications").Create(&notif).Error; err != nil {
			return err
		}
	}
	return nil
}

// ==========================================
// 6. POST /api/notifications/send-response (ส่งหา พนักงาน)
// ==========================================
func (r *NotificationRepo) CreateResponseNotification(managerID, reqType, requestNumber, status string) error {
	prefix := "LEV"
	tableName := "leave_requests"
	if strings.HasPrefix(requestNumber, "REQ") {
		prefix = "REQ"
		tableName = "attendance_requests"
	}

	idStr := strings.TrimLeft(strings.TrimPrefix(requestNumber, prefix), "0")
	reqID, _ := strconv.Atoi(idStr)

	// ดึงข้อมูลคำขอ
	var req struct {
		UserID    string    `gorm:"column:user_id"`
		DateFrom  time.Time `gorm:"column:date_from"`
		DateTo    time.Time `gorm:"column:date_to"`
		LeaveType string    `gorm:"column:leave_type"`
	}
	
	query := r.db.Table(tableName).Select("user_id, date_from, date_to").Where("id = ?", reqID)
	if prefix == "LEV" {
		query.Select("user_id, date_from, date_to, leave_type")
	}
	if err := query.Scan(&req).Error; err != nil || req.UserID == "" {
		return errors.New("request not found")
	}

	// ดึงชื่อบอสที่กดอนุมัติ
	var managerName string
	r.db.Table("user_info").Where("user_id = ?", managerID).Select("fullname_thai").Scan(&managerName)
	if managerName == "" {
		managerName = "ผู้บังคับบัญชา"
	}

	// สร้าง Title และ Message ตาม Format
	dateStr := formatThaiDateRange(req.DateFrom, req.DateTo)
	statusTh := "ถูกอนุมัติ"
	if status == "REJECTED" {
		statusTh = "ถูกปฏิเสธ"
	} else if status == "CANCELED" {
		statusTh = "ถูกยกเลิก"
	}

	var title, message string
	if reqType == "LEAVE_REQUEST" {
		title = fmt.Sprintf("คำขอลางาน%s", statusTh)
		
		leaveName := "ลางาน"
		r.db.Table("leave_types").Where("name_en = ?", req.LeaveType).Select("name_th").Scan(&leaveName)
		
		// 🌟 [NEW] เช็คและเติมคำว่า "ลา" ในฝั่งส่งกลับหาพนักงานด้วยเหมือนกัน
		if !strings.HasPrefix(leaveName, "ลา") {
			leaveName = "ลา" + leaveName
		}
		
		message = fmt.Sprintf("คำขอ%s%s %sโดย %s", leaveName, dateStr, statusTh, managerName)
	} else {
		title = fmt.Sprintf("คำขอเวลาเข้า-ออกงาน%s", statusTh)
		message = fmt.Sprintf("คำขออนุมัติเวลาเข้า-ออกงาน%s %sโดย %s", dateStr, statusTh, managerName)
	}

	notif := Notification{
		ID:            "notif_" + uuid.New().String()[:8],
		UserID:        req.UserID, 
		Title:         title,
		Message:       message,
		IsRead:        false,
		Type:          reqType,
		Status:        status,
		RequestNumber: requestNumber,
		CreatedAt:     time.Now(),
	}

	return r.db.Table("notifications").Create(&notif).Error
}