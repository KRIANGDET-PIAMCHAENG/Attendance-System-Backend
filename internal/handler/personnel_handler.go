package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"my-app/internal/repository" // ⚠️ แก้ path นี้นะครับ
)

type PersonnelHandler struct {
	repo *repository.PersonnelRepo
}

func NewPersonnelHandler(repo *repository.PersonnelRepo) *PersonnelHandler {
	return &PersonnelHandler{repo: repo}
}

func (h *PersonnelHandler) GetPending(c *gin.Context) {
	id := c.Query("id")
	res, err := h.repo.GetPending(id)
	if err != nil || res == nil {
		c.JSON(http.StatusOK, gin.H{"pending": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pending": res})
}

func (h *PersonnelHandler) GetRecent(c *gin.Context) {
	id := c.Query("id")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	res, err := h.repo.GetRecent(id, startDate, endDate)
	if err != nil || res == nil {
		c.JSON(http.StatusOK, gin.H{"recent": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recent": res})
}

func (h *PersonnelHandler) GetFilterRange(c *gin.Context) {
	id := c.Query("id")
	start, end, err := h.repo.GetFilterRange(id)
	
	if err != nil || start.IsZero() || end.IsZero() {
		// ถ้าคนนี้ไม่เคยลาเลย ส่งค่า default กลับไปกันแอพพัง
		c.JSON(http.StatusOK, gin.H{
			"start": "2020-01-01T00:00:00.000Z",
			"end":   "2030-12-31T00:00:00.000Z",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	})
}

func (h *PersonnelHandler) GetDetail(c *gin.Context) {
	reqIDStr := c.Query("request-id")
	idStr := strings.TrimPrefix(reqIDStr, "LEV")
	reqID, err := strconv.Atoi(idStr)
	
	if err != nil || reqID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ไม่ถูกต้อง"})
		return
	}

	res, err := h.repo.GetDetail(reqID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลคำขอ"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *PersonnelHandler) GetUsers(c *gin.Context) {
	res, err := h.repo.GetUsers()
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