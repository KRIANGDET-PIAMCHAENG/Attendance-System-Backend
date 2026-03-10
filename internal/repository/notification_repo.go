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

// 5. POST /api/notifications/send-request (ส่งหา หัวหน้า)
func (r *NotificationRepo) CreateRequestNotification(reqType, requestNumber string) error {
	// ตัดตัวอักษรเพื่อหา ID ของคำขอ (เช่น LEV0000000065012 -> 65012)
	prefix := "LEV"
	tableName := "leave_requests"
	if strings.HasPrefix(requestNumber, "REQ") {
		prefix = "REQ"
		tableName = "attendance_requests"
	}

	idStr := strings.TrimLeft(strings.TrimPrefix(requestNumber, prefix), "0")
	reqID, _ := strconv.Atoi(idStr)

	// หาว่าใครเป็นคนส่งคำขอ
	var ownerID string
	r.db.Table(tableName).Where("id = ?", reqID).Select("user_id").Scan(&ownerID)
	if ownerID == "" {
		return errors.New("request not found")
	}

	// หาหัวหน้าของคนส่งคำขอ (ดึงมาจากตาราง subordinate_manager_roles)
	var managerIDs []string
	r.db.Table("subordinate_manager_roles smr").
		Select("ur.user_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("smr.subordinate_id = ?", ownerID).
		Pluck("user_id", &managerIDs)

	// สร้างแจ้งเตือนส่งให้หัวหน้าทุกคนที่มีสิทธิ์อนุมัติ
	for _, managerID := range managerIDs {
		notif := Notification{
			ID:            "notif_" + uuid.New().String()[:8],
			UserID:        managerID,
			Title:         "มีคำขอใหม่รอการอนุมัติ",
			Message:       fmt.Sprintf("มีคำขอ %s เลขที่ %s รอการตรวจสอบจากคุณ", reqType, requestNumber),
			IsRead:        false,
			Type:          reqType,
			Status:        "PENDING",
			RequestNumber: requestNumber,
			CreatedAt:     time.Now(),
		}
		r.db.Table("notifications").Create(&notif)
	}
	return nil
}

// 6. POST /api/notifications/send-response (ส่งหา พนักงาน)
func (r *NotificationRepo) CreateResponseNotification(reqType, requestNumber, status string) error {
	prefix := "LEV"
	tableName := "leave_requests"
	if strings.HasPrefix(requestNumber, "REQ") {
		prefix = "REQ"
		tableName = "attendance_requests"
	}

	idStr := strings.TrimLeft(strings.TrimPrefix(requestNumber, prefix), "0")
	reqID, _ := strconv.Atoi(idStr)

	// หาว่าใครเป็นเจ้าของคำขอ
	var ownerID string
	r.db.Table(tableName).Where("id = ?", reqID).Select("user_id").Scan(&ownerID)
	if ownerID == "" {
		return errors.New("request not found")
	}

	statusTh := "ได้รับการอนุมัติ"
	if status == "REJECTED" {
		statusTh = "ถูกปฏิเสธ"
	}

	notif := Notification{
		ID:            "notif_" + uuid.New().String()[:8],
		UserID:        ownerID, // ส่งกลับไปหาพนักงาน
		Title:         fmt.Sprintf("คำขอ%s", statusTh),
		Message:       fmt.Sprintf("คำขอเลขที่ %s %s เรียบร้อยแล้ว", requestNumber, statusTh),
		IsRead:        false,
		Type:          reqType,
		Status:        status,
		RequestNumber: requestNumber,
		CreatedAt:     time.Now(),
	}

	// 🌟 เติม .Error ต่อท้ายเข้าไปครับ
	return r.db.Table("notifications").Create(&notif).Error
}