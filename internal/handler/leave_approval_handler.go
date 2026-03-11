package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"my-app/internal/repository" // เปลี่ยนชื่อ package เป็นของโปรเจกต์คุณ
)

type LeaveApprovalHandler struct {
	repo      *repository.LeaveApprovalRepo
	notifRepo *repository.NotificationRepo // 🌟 ต้องเรียกใช้ Notification
}

func NewLeaveApprovalHandler(repo *repository.LeaveApprovalRepo, notifRepo *repository.NotificationRepo) *LeaveApprovalHandler {
	return &LeaveApprovalHandler{repo: repo, notifRepo: notifRepo}
}

func (h *LeaveApprovalHandler) GetPendingSummary(c *gin.Context) {
	managerID := c.GetString("user_id")
	res, err := h.repo.GetPendingSummary(managerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"pending": res}})
}

func (h *LeaveApprovalHandler) GetRecent(c *gin.Context) {
	managerID := c.GetString("user_id")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	res, err := h.repo.GetRecent(managerID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"recent": res}})
}

func (h *LeaveApprovalHandler) GetFilterRange(c *gin.Context) {
	managerID := c.GetString("user_id")
	res, err := h.repo.GetFilterRange(managerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *LeaveApprovalHandler) GetUserDetail(c *gin.Context) {
	managerID := c.GetString("user_id")
	targetUserID := c.Query("user-id")

	res, err := h.repo.GetUserDetail(managerID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

// 🌟 แก้ Handler: ดึง Base URL อัตโนมัติเหมือนกัน
func (h *LeaveApprovalHandler) GetRequestDetail(c *gin.Context) {
    managerID := c.GetString("user_id")
    reqIDStr := c.Query("request-id")
    reqIDStr = strings.TrimLeft(strings.TrimPrefix(reqIDStr, "LEV"), "0")
    reqID, _ := strconv.Atoi(reqIDStr)

    scheme := "http"
    if c.Request.TLS != nil {
        scheme = "https"
    }
    baseURL := scheme + "://" + c.Request.Host + "/"

    // 🌟 ส่ง baseURL ไปด้วย
    res, err := h.repo.GetRequestDetail(managerID, reqID, baseURL)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *LeaveApprovalHandler) ApproveReject(c *gin.Context) {
	managerID := c.GetString("user_id")
	reqIDStr := c.Param("id") // รับจาก URL เช่น /api/leave-approval/LEV000000000030
	
	fullReqID := reqIDStr // เก็บไว้ส่ง Noti
	reqIDStr = strings.TrimLeft(strings.TrimPrefix(reqIDStr, "LEV"), "0")
	reqID, _ := strconv.Atoi(reqIDStr)

	status := c.PostForm("status")
	reason := c.PostForm("reason")

	// 🌟 จัดการ File Upload (ลายเซ็น)
	var signaturePath string
	file, err := c.FormFile("signature-approval")
	if err == nil {
		now := time.Now()
		uploadDir := fmt.Sprintf("uploads/signatures/%04d/%02d", now.Year(), now.Month())
		os.MkdirAll(uploadDir, os.ModePerm)

		ext := filepath.Ext(file.Filename)
		newFileName := uuid.New().String() + ext
		signaturePath = filepath.Join(uploadDir, newFileName)

		if err := c.SaveUploadedFile(file, signaturePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save signature"})
			return
		}
	}

	// 🌟 บันทึกลง Database
	err = h.repo.ApproveRejectRequest(managerID, reqID, status, reason, signaturePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 🌟 [ทีเด็ด] ยิง Notification กลับไปหาพนักงานทันที!
	_ = h.notifRepo.CreateResponseNotification(managerID, "LEAVE_REQUEST", fullReqID, strings.ToUpper(status))

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}