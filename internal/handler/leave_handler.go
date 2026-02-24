package handler

import (
	"net/http"
	"path/filepath"
	"time"

	"my-app/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"
)

type LeaveHandler struct {
	repo *repository.UserRepo
}

func NewLeaveHandler(repo *repository.UserRepo) *LeaveHandler {
	return &LeaveHandler{repo: repo}
}

func (h *LeaveHandler) CreateLeaveRequest(c *gin.Context) {
	var req repository.CreateLeaveRequest

	// 1. Bind ข้อมูล Text (leave-type, date-from, remark, etc.)
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Form Data", "details": err.Error()})
		return
	}

	// 2. ดึง User ID ออกมาจาก Context (JWT)
	userID := c.MustGet("user_id").(string)

	// ==========================================
	// 🌟 3. จัดการไฟล์ลายเซ็น (Signature)
	// ==========================================
	var signaturePath *string
	sigFile, err := c.FormFile("signature")

	// ถ้ามีการแนบไฟล์ลายเซ็นมา
	if err == nil && sigFile != nil {
		uploadDir := "uploads/signatures/" + time.Now().Format("2006/01")
		os.MkdirAll(uploadDir, os.ModePerm)

		ext := filepath.Ext(sigFile.Filename)
		newSigName := uuid.New().String() + ext
		dst := filepath.Join(uploadDir, newSigName)

		if err := c.SaveUploadedFile(sigFile, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกไฟล์ลายเซ็นไม่สำเร็จ"})
			return
		}
		signaturePath = &dst // เก็บ Path ไว้ส่งให้ Database
	}

	// ==========================================
	// 💾 4. บันทึกข้อมูลใบลาลง DB (ส่ง signaturePath ไปด้วย)
	// ==========================================
	leaveID, err := h.repo.SaveLeaveRequest(userID, req, signaturePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	// ==========================================
	// 📂 5. จัดการไฟล์แนบ (Files Array)
	// ==========================================
	form, err := c.MultipartForm()
	var filesCount int

	if err == nil && form != nil {
		files := form.File["files"]
		filesCount = len(files)

		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			newFileName := uuid.New().String() + ext

			uploadDir := "uploads/leave_requests/" + time.Now().Format("2006/01")
			if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
				continue
			}

			dst := filepath.Join(uploadDir, newFileName)

			if err := c.SaveUploadedFile(file, dst); err != nil {
				continue
			}

			h.repo.SaveLeaveAttachment(leaveID, dst, file.Filename)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Leave request created",
		"files_count": filesCount,
	})
}