package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"my-app/internal/repository" // ⚠️ อย่าลืมแก้ Path
)

type PersonnelHandler struct {
	repo *repository.PersonnelRepo
}

func NewPersonnelHandler(repo *repository.PersonnelRepo) *PersonnelHandler {
	return &PersonnelHandler{repo: repo}
}

// Helper function ดึง user_id จาก JWT Token ของคนส่ง request
func getManagerID(c *gin.Context) string {
	// ⚠️ ถ้า JWTMiddleware ลูกพี่เซ็ตคีย์ชื่ออื่น (เช่น "userID") ให้เปลี่ยนให้ตรงนะครับ
	return c.GetString("user_id") 
}

func (h *PersonnelHandler) GetPending(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")

	res, err := h.repo.GetPending(managerID, personnelID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "pending": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pending": res})
}

func (h *PersonnelHandler) GetRecent(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	res, err := h.repo.GetRecent(managerID, personnelID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "recent": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recent": res})
}

func (h *PersonnelHandler) GetFilterRange(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")

	start, end, err := h.repo.GetFilterRange(managerID, personnelID)
	if err != nil || start.IsZero() || end.IsZero() {
		c.JSON(http.StatusForbidden, gin.H{"error": "ไม่มีสิทธิ์ หรือไม่พบข้อมูล"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	})
}

func (h *PersonnelHandler) GetDetail(c *gin.Context) {
	managerID := getManagerID(c)
	reqIDStr := c.Query("request-id")
	
	// ✂️ หั่นคำว่า LEV ทิ้ง เพื่อให้เหลือแต่เลขไปค้นใน Database
	idStr := strings.TrimPrefix(reqIDStr, "LEV")
	reqID, err := strconv.Atoi(idStr)
	
	if err != nil || reqID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID คำขอไม่ถูกต้อง (ต้องเป็น LEV...)"})
		return
	}

	res, err := h.repo.GetDetail(managerID, reqID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *PersonnelHandler) GetUsers(c *gin.Context) {
	managerID := getManagerID(c)

	res, err := h.repo.GetUsers(managerID)
	if err != nil || res == nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *PersonnelHandler) GetPermissionLevel(c *gin.Context) {
	// 1. ดึง ID ของคนถูกดูจาก Query Param
	targetID := c.Query("id")
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ id ของพนักงาน"})
		return
	}

	// 2. ดึง ID ของคนกดดู จาก Context (ที่ JWTMiddleware ฝังเอาไว้ให้)
	// ⚠️ หมายเหตุ: ตรง "user_id" ลูกพี่ต้องเปลี่ยนให้ตรงกับ Key ที่ JWTMiddleware ของลูกพี่ตั้งไว้นะครับ 
	// (บางคนใช้ "userID", "id", หรือดึงจาก Struct) สมมติว่า middleware ใช้ c.Set("user_id", token.UserID)
	requesterID := c.GetString("user_id") 
	
	// ถ้าหา requesterID จาก token ไม่เจอ (เผื่อกันเหนียว)
	if requesterID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งานใน Token"})
		return
	}

	// 3. เรียก Repo ไปเช็คสิทธิ์
	level, err := h.repo.CheckApprovalPermission(requesterID, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการตรวจสอบสิทธิ์"})
		return
	}

	// 4. ส่งผลลัพธ์กลับไปตามฟอร์แมตเป๊ะๆ
	c.JSON(http.StatusOK, gin.H{
		"permission-level": level,
	})
}

func (h *PersonnelHandler) GetPersonnelData(c *gin.Context) {
	// ดึง ID ของคนยิง (Manager) ออกมาจาก JWT Token
	managerID := getManagerID(c)
	// ดึง ID ลูกน้องที่ต้องการดู จาก URL Query String (?id=xxx)
	personnelID := c.Query("id")

	if personnelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ id ของพนักงาน"})
		return
	}

	// เรียก Repo เพื่อดึงข้อมูลพร้อมเช็คสิทธิ์
	res, err := h.repo.GetPersonnelData(managerID, personnelID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// โยนผลลัพธ์กลับแบบ 200 OK
	c.JSON(http.StatusOK, res)
}

// 🌟 Statistic (Working Hours)
func (h *PersonnelHandler) GetManagerWorkingHoursStatistic(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	if personnelID == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ id"}); return }
	res, err := h.repo.GetManagerWorkingHoursStatistic(managerID, personnelID)
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, res)
}

// 🌟 Statistic & Attendance (Filter Range)
func (h *PersonnelHandler) GetManagerStatFilterRange(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	if personnelID == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ id"}); return }
	res, err := h.repo.GetManagerStatFilterRange(managerID, personnelID)
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, res)
}

// 🌟 Attendance Requests (Pending)
func (h *PersonnelHandler) GetAttReqPending(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	res, err := h.repo.GetAttReqPending(managerID, personnelID)
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"pending": res})
}

// 🌟 Attendance Requests (Recent)
func (h *PersonnelHandler) GetAttReqRecent(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	res, err := h.repo.GetAttReqRecent(managerID, personnelID, c.Query("startDate"), c.Query("endDate"))
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"recent": res})
}

// 🌟 Attendance Requests (Filter Range)
func (h *PersonnelHandler) GetAttReqFilterRange(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	res, err := h.repo.GetAttReqFilterRange(managerID, personnelID)
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, res)
}

// 🌟 Attendance Requests (Detail)
func (h *PersonnelHandler) GetAttReqDetail(c *gin.Context) {
	managerID := getManagerID(c)
	reqIDStr := c.Query("request-id")
	idStr := strings.TrimPrefix(reqIDStr, "REQ") // หั่นคำว่า REQ ทิ้ง
	reqID, err := strconv.Atoi(idStr)
	if err != nil || reqID == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบ ID คำขอไม่ถูกต้อง"}); return }

	res, err := h.repo.GetAttReqDetail(managerID, reqID)
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, res)
}

// 🌟 Attendance History
func (h *PersonnelHandler) GetAttendanceHistory(c *gin.Context) {
	managerID := getManagerID(c)
	personnelID := c.Query("id")
	res, err := h.repo.GetAttendanceHistory(managerID, personnelID, c.Query("startDate"), c.Query("endDate"))
	if err != nil { c.JSON(http.StatusForbidden, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, res)
}



// 🌟 Get Statistic (Manager ดูสถิติลูกน้อง)
func (h *PersonnelHandler) GetStatistic(c *gin.Context) {
	// 1. ดึง ID ของ Manager จาก Token 
	// (ใช้ฟังก์ชันเดิมที่ลูกพี่มีอยู่แล้ว เช่น c.GetString("user_id") หรือ getManagerID(c))
	managerID := getManagerID(c) 
	
	// 2. ดึง ID ลูกน้องจาก Query Parameter (?id=...)
	personnelID := c.Query("id")
	if personnelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ id ของพนักงาน"})
		return
	}

	// 3. รับค่า Year (ค.ศ.) จาก Query Parameter (?year=...)
	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)
	
	// ถ้าปีที่ส่งมาพัง หรือไม่ยอมส่งมา ให้ Fallback เป็นปีปัจจุบันอัตโนมัติ
	if err != nil || year == 0 {
		year = time.Now().Year() 
	}

	// 4. ส่งไปคำนวณที่ Repository
	res, err := h.repo.GetStatistic(managerID, personnelID, year)
	if err != nil {
		// ถ้ามี Error เช่น ไม่มีสิทธิ์ ให้ส่ง 403 Forbidden กลับไป
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 5. ส่ง JSON หล่อๆ กลับไปให้ Frontend
	c.JSON(http.StatusOK, res)
}