package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"my-app/internal/repository"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

	if err == nil && form != nil {
		files := form.File["files"]

		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			newFileName := uuid.New().String() + ext

			// 🌟 1. ดึงชนิดไฟล์ (ตัดจุด . ออกให้เหลือแค่ pdf, jpg)
			fileType := strings.TrimPrefix(ext, ".")

			// 🌟 2. ดึงขนาดไฟล์จริง (ของแท้ 100% จาก multipart)
			fileSize := file.Size

			uploadDir := "uploads/leave_requests/" + time.Now().Format("2006/01")
			if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
				continue
			}

			dst := filepath.Join(uploadDir, newFileName)

			if err := c.SaveUploadedFile(file, dst); err != nil {
				continue
			}

			// 🌟 3. อัปเดตการเรียกใช้ Repo โดยส่ง fileType และ fileSize พ่วงไปด้วย
			h.repo.SaveLeaveAttachment(leaveID, dst, file.Filename, fileType, fileSize)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		// ใช้ fmt.Sprintf เพื่อเติม LEV และ 0 ให้ครบ 12 หลักตามสเปคเป๊ะๆ
		"request-id": fmt.Sprintf("LEV%012d", leaveID),
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

func (h *LeaveHandler) GetRecentLeaves(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	records, err := h.repo.GetRecentLeaves(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	var recentList []map[string]interface{}
	for _, r := range records {
		thaiTime := r.DateStart.In(loc)

		// 🚩 ตรรกะการกำหนด Status String
		finalStatus := r.Status
		if r.Status == "pending" {
			finalStatus = "overdue" // ถ้าหลุดมาจาก Query ข้างบนแล้วยังเป็น pending แปลว่ามัน overdue
		}

		recentList = append(recentList, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", r.ID),
			"leave-type": r.LeaveType,
			"date-start": thaiTime.Format("2006-01-02T15:04:05.000+07:00"),
			"status":     finalStatus, // ส่งค่า 'approved', 'rejected' หรือ 'overdue'
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

func (h *LeaveHandler) GetLeaveDetail(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// 1. รับ ID จาก Query (เช่น ?request-id=LEV000000000015)
	reqIDStr := c.Query("request-id")
	if len(reqIDStr) <= 3 || reqIDStr[:3] != "LEV" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID ไม่ถูกต้อง"})
		return
	}

	// 2. ตัด "LEV" ออก และแปลง 000000000015 เป็นตัวเลข
	leaveID, err := strconv.Atoi(reqIDStr[3:])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ใบลาไม่ถูกต้อง"})
		return
	}

	// 3. ดึงข้อมูลจาก Repository
	detail, attachments, err := h.repo.GetLeaveDetail(userID, leaveID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลใบลา"})
		return
	}

	// 4. ตั้งค่า Timezone ไทย
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	// ตรรกะเช็คสถานะ overdue
	finalStatus := detail.Status
	if finalStatus == "pending" && detail.DateFrom.Before(time.Now()) {
		finalStatus = "overdue"
	}

    scheme := "http"
    if c.Request.TLS != nil {
        scheme = "https"
    }
    baseURL := fmt.Sprintf("%s://%s/", scheme, c.Request.Host)

	// 5. ปั้นข้อมูลไฟล์แนบ
	var evidenceFiles []map[string]interface{}
	for _, att := range attachments {
		evidenceFiles = append(evidenceFiles, map[string]interface{}{
			"file-name": att.OriginalName,
			"file-url":  baseURL + att.FilePath, // อนาคตอาจจะเอา Domain มาต่อหน้า FilePath ตรงนี้
			"file-type": att.FileType,
			"file-size": att.FileSize,
		})
	}
	if evidenceFiles == nil {
		evidenceFiles = []map[string]interface{}{}
	}

	// 6. ปั้นข้อมูลการอนุมัติ (เช็ค nil pointer ด้วย)
	approveDetail := map[string]interface{}{
		"status":       finalStatus,
		"approve-role": detail.ApproveRole,
		"approver":     detail.Approver,
		"reason":       detail.ApproveReason,
		"approve-date": nil,
	}
	if detail.ApproveDate != nil {
		approveDetail["approve-date"] = detail.ApproveDate.In(loc).Format("2006-01-02T15:04:05.000+07:00")
	}

	// 7. ส่ง JSON กลับไปให้ Frontend
	response := gin.H{
		"request-detail": map[string]interface{}{
			"leave-type":        detail.LeaveType,
			"date-from":         detail.DateFrom.In(loc).Format("2006-01-02T15:04:05.000+07:00"),
			"date-to":           detail.DateTo.In(loc).Format("2006-01-02T15:04:05.000+07:00"),
			"from-date-morning": detail.FromDateMorning,
			"to-date-morning":   detail.ToDateMorning,
			"remark":            detail.Remark,
			"evidence-files":    evidenceFiles,
			"request-date":      detail.CreatedAt.In(loc).Format("2006-01-02T15:04:05.000+07:00"),
		},
		"approve-detail": approveDetail,
	}

	c.JSON(http.StatusOK, response)
}

func (h *LeaveHandler) GetLeaveInfo(c *gin.Context) {
	// 1. รับ UserID จาก Context (JWT Token)
	userID := c.MustGet("user_id").(string)

	// 2. รับค่าประเภทการลาจาก Query (เช่น ?leave-type=sick)
	leaveType := c.Query("leave-type")
	if leaveType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ leave-type"})
		return
	}

	// 3. เรียกใช้งาน Repository เพื่อดึงโควต้าของปีงบปัจจุบัน
	used, max, err := h.repo.GetLeaveBalanceInfo(userID, leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 4. ส่งข้อมูลกลับ (ตัวเลข Float จะถูกแปลงเป็น JSON Number เช่น 10.5 และ 60)
	c.JSON(http.StatusOK, gin.H{
		"used": used,
		"max":  max,
	})
}

func (h *LeaveHandler) ResendLeaveRequest(c *gin.Context) {
	// 1. ดึง User ID
	userID := c.MustGet("user_id").(string)

	// รับค่าจาก Text Form
	reqIDStr := c.PostForm("request-id")
	remark := c.PostForm("remark")

	// 🌟 แก้ตรงนี้: เช็คทั้งแบบมีและไม่มี []
	oldFiles := c.PostFormArray("old-files")
	if len(oldFiles) == 0 {
		oldFiles = c.PostFormArray("old-files[]") // ดักเผื่อ Dio เติม [] มาให้
	}

	// ตัดคำว่า "LEV" ออกแล้วแปลงเป็นตัวเลข (เช่น LEV000000000012 -> 12)
	idStr := strings.TrimPrefix(reqIDStr, "LEV")
	leaveID, err := strconv.Atoi(idStr)
	if err != nil || leaveID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ request-id ไม่ถูกต้อง"})
		return
	}

	// ==========================================
	// 3. จัดการไฟล์ลายเซ็นใหม่ (ถ้ามีการส่งมาทับของเดิม)
	// ==========================================
	var signaturePath *string
	sigFile, err := c.FormFile("signature")
	if err == nil && sigFile != nil {
		uploadDir := "uploads/signatures/" + time.Now().Format("2006/01")
		os.MkdirAll(uploadDir, os.ModePerm)

		ext := filepath.Ext(sigFile.Filename)
		newSigName := uuid.New().String() + ext
		dst := filepath.Join(uploadDir, newSigName)

		if err := c.SaveUploadedFile(sigFile, dst); err == nil {
			signaturePath = &dst
		}
	}

	// ==========================================
	// 4. จัดการไฟล์แนบใหม่ (New Files)
	// ==========================================
	form, err := c.MultipartForm()
	var newAttachments []repository.NewAttachment

	if err == nil && form != nil {
		files := form.File["files"]
		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			newFileName := uuid.New().String() + ext
			fileType := strings.TrimPrefix(ext, ".")

			uploadDir := "uploads/leave_requests/" + time.Now().Format("2006/01")
			os.MkdirAll(uploadDir, os.ModePerm)
			dst := filepath.Join(uploadDir, newFileName)

			if err := c.SaveUploadedFile(file, dst); err == nil {
				newAttachments = append(newAttachments, repository.NewAttachment{
					FilePath:     dst,
					OriginalName: file.Filename,
					FileType:     fileType,
					FileSize:     file.Size,
				})
			}
		}
	}

	// ==========================================
	// 5. ส่งให้ Repository บันทึกข้อมูล
	// ==========================================
	if err := h.repo.ResendLeaveRequest(userID, leaveID, remark, oldFiles, signaturePath, newAttachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถอัปเดตข้อมูลใบลาได้", "details": err.Error()})
		return
	}

	// ส่ง 200 OK กลับไป
	c.JSON(http.StatusOK, gin.H{
		"message":    "ยื่นใบลาใหม่อีกครั้งสำเร็จ",
		"request-id": reqIDStr,
	})
}

func (h *LeaveHandler) CancelLeaveRequest(c *gin.Context) {
    // 1. ดึง User ID
    userID := c.MustGet("user_id").(string)

    // 2. รับค่า Request ID (รองรับทั้ง JSON Body และ Form-Data)
    var req struct {
        RequestID string `json:"request-id" form:"request-id"`
    }
    
    if err := c.ShouldBind(&req); err != nil || req.RequestID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง หรือไม่ได้ส่ง request-id"})
        return
    }

    // 3. ตัดคำว่า "LEV" ออก
    idStr := strings.TrimPrefix(req.RequestID, "LEV")
    leaveID, err := strconv.Atoi(idStr)
    if err != nil || leaveID == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ request-id ไม่ถูกต้อง"})
        return
    }

    // 4. ส่งให้ Repository เปลี่ยนสถานะ
    if err := h.repo.CancelLeaveRequest(userID, leaveID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถยกเลิกใบลาได้", "details": err.Error()})
        return
    }

    // 5. ส่งผลลัพธ์สำเร็จ
    c.JSON(http.StatusOK, gin.H{
        "message": "ยกเลิกใบลาสำเร็จ", // เปลี่ยนข้อความให้ซอฟต์ลง
    })
}
