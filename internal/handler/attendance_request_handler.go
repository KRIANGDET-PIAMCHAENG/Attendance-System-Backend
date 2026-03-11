package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"strconv"
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

	reqIDStr := fmt.Sprintf("REQ%011d", req.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"request-id":      reqIDStr, // ส่งรหัสกลับไปให้ Flutter
	})
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
			"status":   r.Status ,
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


// Helper Function เอาไว้ดึง BaseURL (ใช้ใน Detail)
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/", scheme, c.Request.Host)
}

func (h *AttendanceReqHandler) GetAttendanceDetail(c *gin.Context) {
    userID := c.MustGet("user_id").(string)
    reqIDStr := c.Query("id")

    idStr := strings.TrimPrefix(reqIDStr, "REQ")
    reqID, err := strconv.Atoi(idStr)
    if err != nil || reqID == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID ไม่ถูกต้อง"})
        return
    }

    request, approverName, expectedRole, err := h.repo.GetAttendanceDetail(userID, reqID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงข้อมูลไม่สำเร็จ"})
        return
    }

    baseURL := getBaseURL(c)
    var evidenceFiles []map[string]interface{}
    for _, file := range request.Attachments {
        fixedPath := strings.ReplaceAll(file.FilePath, "\\", "/")
        
        evidenceFiles = append(evidenceFiles, map[string]interface{}{
            "file-name": file.OriginalName,
            "file-size": file.FileSize,
            "file-type": file.FileType,
            "file-url":  baseURL + fixedPath,
        })
    }

    if evidenceFiles == nil {
        evidenceFiles = []map[string]interface{}{}
    }

    approveDetail := map[string]interface{}{
        "status":       request.Status, 
        "approve-role": expectedRole, 
    }

    if request.Approval != nil {
        approveDetail["approver"] = approverName
        approveDetail["reason"] = request.Approval.Reason
        // 🌟 เอา .UTC() ออก และตัด Z ทิ้ง
        approveDetail["approve-date"] = request.Approval.CreatedAt.Format("2006-01-02T15:04:05")
    } else {
        approveDetail["approver"] = ""
        approveDetail["reason"] = ""
        approveDetail["approve-date"] = ""
    }

    c.JSON(http.StatusOK, gin.H{
        "request-detail": map[string]interface{}{
            // 🌟 เอา .UTC() ออก และตัด Z ทิ้ง
            "date-from":      request.DateFrom.Format("2006-01-02T15:04:05"),
            "date-to":        request.DateTo.Format("2006-01-02T15:04:05"),
            "time-start":     request.StartTime, 
            "time-end":       request.EndTime,   
            "remark":         request.Remark,
            "evidence-files": evidenceFiles,
        },
        "approve-detail": approveDetail,
    })
}

// 6. Delete Request (รับค่าจาก JSON Body: {"id": "REQ..."})
func (h *AttendanceReqHandler) DeleteAttendanceRequest(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var req struct {
		ID string `json:"id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูล Body ไม่ถูกต้อง"})
		return
	}

	idStr := strings.TrimPrefix(req.ID, "REQ")
	reqID, err := strconv.Atoi(idStr)
	if err != nil || reqID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID ไม่ถูกต้อง"})
		return
	}

	if err := h.repo.DeleteAttendanceRequest(userID, reqID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ยกเลิกคำขอสำเร็จ"})
}

// 7. Resend Request (รับเป็น Multipart Form-Data)
func (h *AttendanceReqHandler) ResendAttendanceRequest(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	reqIDStr := c.PostForm("id")
	remark := c.PostForm("remark")

	oldFiles := c.PostFormArray("old-files")
	if len(oldFiles) == 0 {
		oldFiles = c.PostFormArray("old-files[]") // ดักเคส Dio ชอบแอบเติม [] ให้
	}

	idStr := strings.TrimPrefix(reqIDStr, "REQ")
	reqID, err := strconv.Atoi(idStr)
	if err != nil || reqID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID ไม่ถูกต้อง"})
		return
	}

	// 7.1 จัดการไฟล์ลายเซ็นใหม่ (ถ้ามี)
	var signaturePath *string
	sigFile, err := c.FormFile("signature")
	if err == nil && sigFile != nil {
		uploadDir := "uploads/signatures/" + time.Now().Format("2006/01")
		os.MkdirAll(uploadDir, os.ModePerm)
		dst := filepath.Join(uploadDir, uuid.New().String()+filepath.Ext(sigFile.Filename))
		if err := c.SaveUploadedFile(sigFile, dst); err == nil {
			signaturePath = &dst
		}
	}

	// 7.2 จัดการไฟล์แนบใหม่ (ถ้ามี)
	form, err := c.MultipartForm()
	var newAttachments []repository.NewAttendanceAttachment
	if err == nil && form != nil {
		files := form.File["files"]
		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			uploadDir := "uploads/attendance_requests/" + time.Now().Format("2006/01")
			os.MkdirAll(uploadDir, os.ModePerm)
			dst := filepath.Join(uploadDir, uuid.New().String()+ext)

			if err := c.SaveUploadedFile(file, dst); err == nil {
				newAttachments = append(newAttachments, repository.NewAttendanceAttachment{
					FilePath:     dst,
					OriginalName: file.Filename,
					FileType:     strings.TrimPrefix(ext, "."),
					FileSize:     file.Size,
				})
			}
		}
	}

	// 7.3 ส่งต่อให้ Repo
	if err := h.repo.ResendAttendanceRequest(userID, reqID, remark, oldFiles, signaturePath, newAttachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ส่งคำขอใหม่อีกครั้งสำเร็จ"})
}