package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"my-app/internal/repository" // เปลี่ยนเป็น path โปรเจกต์คุณ
)

type AttendanceReqHandler struct {
	repo *repository.UserRepo // ใช้ UserRepo ตามโครงสร้างเดิมของคุณ
}

func NewAttendanceReqHandler(repo *repository.UserRepo) *AttendanceReqHandler {
	return &AttendanceReqHandler{repo: repo}
}

// 1. Create Request
func (h *AttendanceReqHandler) CreateTimeRequest(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	dateFrom, _ := time.Parse(time.RFC3339, c.PostForm("date-from"))
	dateTo, _ := time.Parse(time.RFC3339, c.PostForm("date-to"))
	startTime := c.PostForm("start-time")
	endTime := c.PostForm("end-time")
	remark := c.PostForm("remark")

	// 1.1 จัดการไฟล์ลายเซ็น
	var signaturePath string
	sigFile, err := c.FormFile("signature")
	if err == nil && sigFile != nil {
		uploadDir := "uploads/signatures/" + time.Now().Format("2006/01")
		os.MkdirAll(uploadDir, os.ModePerm)
		dst := filepath.Join(uploadDir, uuid.New().String()+filepath.Ext(sigFile.Filename))
		if err := c.SaveUploadedFile(sigFile, dst); err == nil {
			signaturePath = dst
		}
	}

	// 1.2 จัดการไฟล์แนบ
	form, err := c.MultipartForm()
	var attachments []repository.NewAttendanceAttachment
	if err == nil && form != nil {
		files := form.File["files"]
		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			dst := filepath.Join("uploads/attendance_requests/"+time.Now().Format("2006/01"), uuid.New().String()+ext)
			os.MkdirAll(filepath.Dir(dst), os.ModePerm)
			if err := c.SaveUploadedFile(file, dst); err == nil {
				attachments = append(attachments, repository.NewAttendanceAttachment{
					FilePath:     dst,
					OriginalName: file.Filename,
					FileType:     strings.TrimPrefix(ext, "."),
					FileSize:     file.Size,
				})
			}
		}
	}

	// 1.3 สร้าง Model และส่งเข้า Repo
	req := repository.AttendanceRequest{
		UserID:        userID,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		StartTime:     startTime,
		EndTime:       endTime,
		Remark:        remark,
		SignaturePath: signaturePath,
		Status:        "pending",
	}

	if err := h.repo.CreateAttendanceRequest(&req, attachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 2. Get Pending
func (h *AttendanceReqHandler) GetPendingRequests(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	requests, err := h.repo.GetPendingAttendanceRequests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var pendingList []map[string]interface{}
	for _, r := range requests {
		pendingList = append(pendingList, map[string]interface{}{
			"id":         fmt.Sprintf("REQ%011d", r.ID), // เติมเลข 0 ให้ครบตาม Format REQ00000000012
			"date-start": r.DateFrom.Format(time.RFC3339),
			"date-end":   r.DateTo.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"pending": pendingList})
}

// 3. Get Recent
func (h *AttendanceReqHandler) GetRecentRequests(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// รับค่า Filter จาก Query Parameters (ตามที่แนะนำไปตอนแรก)
	var startDate, endDate *time.Time
	if startStr := c.Query("startDate"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startDate = &t
		}
	}
	if endStr := c.Query("endDate"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endDate = &t
		}
	}

	requests, err := h.repo.GetRecentAttendanceRequests(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var recentList []map[string]interface{}
	for _, r := range requests {
		recentList = append(recentList, map[string]interface{}{
			"id":         fmt.Sprintf("REQ%011d", r.ID),
			"date-start": r.DateFrom.Format(time.RFC3339),
			"date-end":   r.DateTo.Format(time.RFC3339),
			"approved":   r.Status == "approved", // ถ้า status เป็น approved = true, ถ้าเป็น rejected = false
		})
	}

	c.JSON(http.StatusOK, gin.H{"recent": recentList})
}

// 4. Get Filter Range
func (h *AttendanceReqHandler) GetFilterRange(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	minDate, maxDate, err := h.repo.GetAttendanceFilterRange(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start": minDate.Format(time.RFC3339),
		"end":   maxDate.Format(time.RFC3339),
	})
}