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

type AttendanceApprovalHandler struct {
	repo      *repository.AttendanceApprovalRepo
	notifRepo *repository.NotificationRepo
}

func NewAttendanceApprovalHandler(repo *repository.AttendanceApprovalRepo, notifRepo *repository.NotificationRepo) *AttendanceApprovalHandler {
	return &AttendanceApprovalHandler{repo: repo, notifRepo: notifRepo}
}

func (h *AttendanceApprovalHandler) GetPending(c *gin.Context) {
	managerID := c.GetString("user_id")
	res, err := h.repo.GetPending(managerID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"pending": res}) // JSON ตาม Mock ของคุณ
}

func (h *AttendanceApprovalHandler) GetRecent(c *gin.Context) {
	managerID := c.GetString("user_id")
	res, err := h.repo.GetRecent(managerID, c.Query("startDate"), c.Query("endDate"))
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"recent": res})
}

func (h *AttendanceApprovalHandler) GetFilterRange(c *gin.Context) {
	managerID := c.GetString("user_id")
	res, err := h.repo.GetFilterRange(managerID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *AttendanceApprovalHandler) GetDetail(c *gin.Context) {
	managerID := c.GetString("user_id")
	reqIDStr := c.Query("request-id")
	reqIDStr = strings.TrimLeft(strings.TrimPrefix(reqIDStr, "REQ"), "0")
	reqID, _ := strconv.Atoi(reqIDStr)

	res, err := h.repo.GetRequestDetail(managerID, reqID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *AttendanceApprovalHandler) ApproveReject(c *gin.Context) {
	managerID := c.GetString("user_id")
	reqIDStr := c.Param("id")
	fullReqID := reqIDStr
	reqIDStr = strings.TrimLeft(strings.TrimPrefix(reqIDStr, "REQ"), "0")
	reqID, _ := strconv.Atoi(reqIDStr)

	status := c.PostForm("status")
	reason := c.PostForm("reason")

	var signaturePath string
	file, err := c.FormFile("signature-approval")
	if err == nil {
		now := time.Now()
		uploadDir := fmt.Sprintf("uploads/signatures/%04d/%02d", now.Year(), now.Month())
		os.MkdirAll(uploadDir, os.ModePerm)
		signaturePath = filepath.Join(uploadDir, uuid.New().String()+filepath.Ext(file.Filename))
		c.SaveUploadedFile(file, signaturePath)
	}

	err = h.repo.ApproveRejectRequest(managerID, reqID, status, reason, signaturePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 🌟 ส่ง Notification
	_ = h.notifRepo.CreateResponseNotification(managerID, "ATTENDANCE_REQUEST", fullReqID, strings.ToUpper(status))

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}