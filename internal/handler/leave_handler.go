package handler

import (
    "fmt"
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


    // check ่ว่า overlap ไหม
    isOverlap, err := h.repo.CheckOverlappingLeave(userID, req.DateFrom, req.DateTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบวันที่ไม่ถูกต้อง หรือระบบขัดข้อง", "details": err.Error()})
		return
	}
	if isOverlap {
		c.JSON(http.StatusConflict, gin.H{"error": "ไม่สามารถดำเนินการได้ เนื่องจากมีช่วงเวลาการลาซ้อนทับกับใบลาเดิมที่คุณเคยยื่นไปแล้ว"})
		return
	}

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

// 1. Handler สำหรับ Get Pending
func (h *LeaveHandler) GetPendingLeaves(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	records, err := h.repo.GetPendingLeaves(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 🌟 ตั้งค่า Timezone เป็นเวลาไทย
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60) // Fallback กรณีรันใน Container ที่ไม่มี tzdata
	}

	var pendingList []map[string]interface{}
	for _, r := range records {
		// 🌟 แปลงเวลาเป็น Local (ไทย)
		thaiTime := r.DateStart.In(loc)

		pendingList = append(pendingList, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", r.ID),
			"leave-type": r.LeaveType,
			// 🌟 เปลี่ยน Format เป็น +07:00
			"date-start": thaiTime.Format("2006-01-02T15:04:05.000+07:00"),
		})
	}
	if pendingList == nil {
		pendingList = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"pending": pendingList})
}

// 2. Handler สำหรับ Get Recent
func (h *LeaveHandler) GetRecentLeaves(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	records, err := h.repo.GetRecentLeaves(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 🌟 ตั้งค่า Timezone เป็นเวลาไทย
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	var recentList []map[string]interface{}
	for _, r := range records {
		// 🌟 แปลงเวลาเป็น Local (ไทย)
		thaiTime := r.DateStart.In(loc)

		recentList = append(recentList, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", r.ID),
			"leave-type": r.LeaveType,
			// 🌟 เปลี่ยน Format เป็น +07:00
			"date-start": thaiTime.Format("2006-01-02T15:04:05.000+07:00"),
			"approved":   r.Status == "approved",
		})
	}
	if recentList == nil {
		recentList = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"recent": recentList})
}

// 3. Handler สำหรับ Get Filter Range
func (h *LeaveHandler) GetLeaveFilterRange(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	minDate, maxDate, err := h.repo.GetLeaveFilterRange(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 🌟 ตั้งค่า Timezone เป็นเวลาไทย
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	// กรณีที่ยูสเซอร์คนนี้ไม่เคยยื่นใบลาเลย ให้ Default ส่งปีปัจจุบันไป
	if minDate == nil || maxDate == nil {
		now := time.Now().In(loc) // 🌟 อิงปีจากเวลาไทย
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		end := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, loc)
		
		c.JSON(http.StatusOK, gin.H{
			"start": start.Format("2006-01-02T15:04:05.000+07:00"),
			"end":   end.Format("2006-01-02T15:04:05.000+07:00"),
		})
		return
	}

	// 🌟 แปลงค่าจาก DB เป็นเวลาไทยก่อนส่งกลับ
	c.JSON(http.StatusOK, gin.H{
		"start": minDate.In(loc).Format("2006-01-02T15:04:05.000+07:00"),
		"end":   maxDate.In(loc).Format("2006-01-02T15:04:05.000+07:00"),
	})
}