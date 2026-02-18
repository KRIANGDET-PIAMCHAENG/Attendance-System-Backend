package handler

import (
	"net/http"
	"path/filepath"
	"time"

	"my-app/internal/repository" // ✅ Import repository เพื่อใช้ Struct และ UserRepo

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"
)

// 1. [NEW] ประกาศ Struct LeaveHandler (ที่เคยหายไป)
type LeaveHandler struct {
	repo *repository.UserRepo
}

// 2. [NEW] ประกาศ Constructor (ที่เคยหายไป)
func NewLeaveHandler(repo *repository.UserRepo) *LeaveHandler {
	return &LeaveHandler{repo: repo}
}

// 3. ฟังก์ชันหลัก
func (h *LeaveHandler) CreateLeaveRequest(c *gin.Context) {
    var req repository.CreateLeaveRequest

    // 1. Bind ข้อมูล Text
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Form Data", "details": err.Error()})
        return
    }

    // 2. รับไฟล์
    form, err := c.MultipartForm()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
        return
    }

    files := form.File["files"]

    // -----------------------------------------------------
    // [จุดที่ต้องแก้] 1. ดึง User ID ออกมาจาก Context (JWT)
    // -----------------------------------------------------
    userID := c.MustGet("user_id").(string)

    // -----------------------------------------------------
    // [จุดที่ต้องแก้] 2. ส่ง userID เป็นตัวแปรตัวแรก
    // -----------------------------------------------------
    leaveID, err := h.repo.SaveLeaveRequest(userID, req) 
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
        return
    }

    // Step B: วนลูป Save Files
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

    c.JSON(http.StatusOK, gin.H{"message": "Leave request created", "files_count": len(files)})
}