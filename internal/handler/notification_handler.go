package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"my-app/internal/repository" // เปลี่ยน my-app เป็นชื่อโปรเจกต์ของลูกพี่ด้วยนะครับ
)

type NotificationHandler struct {
	repo *repository.NotificationRepo
}

func NewNotificationHandler(repo *repository.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// 1. GET /notifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id") // ดึงจาก JWT Token
	
	res, err := h.repo.GetNotifications(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// 2. PATCH /notifications/{id}/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetString("user_id")
	notifID := c.Param("id")

	if err := h.repo.MarkAsRead(userID, notifID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 3. PATCH /notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := h.repo.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 4. GET /notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")

	count, err := h.repo.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// 5. POST /api/notifications/send-request
func (h *NotificationHandler) SendRequestNotification(c *gin.Context) {
	var body struct {
		Type          string `json:"type"`
		RequestNumber string `json:"requestNumber"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}

	if err := h.repo.CreateRequestNotification(body.Type, body.RequestNumber); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 6. POST /api/notifications/send-response
func (h *NotificationHandler) SendResponseNotification(c *gin.Context) {
	// 🌟 ดึง ID ของบอสที่กำลังกดอนุมัติ
	managerID := c.GetString("user_id") 

	var body struct {
		Type          string `json:"type"`
		RequestNumber string `json:"requestNumber"`
		Status        string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}

	// 🌟 โยน managerID เข้าไปใน Repo ด้วย
	if err := h.repo.CreateResponseNotification(managerID, body.Type, body.RequestNumber, body.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}