package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"my-app/internal/repository" // ⚠️ แก้ไข path ให้ตรงกับโปรเจกต์ของลูกพี่
)

type StatHandler struct {
	repo *repository.StatRepo
}

func NewStatHandler(repo *repository.StatRepo) *StatHandler {
	return &StatHandler{repo: repo}
}

// 🌟 Get User Statistic
func (h *StatHandler) GetUserStatistic(c *gin.Context) {
	// ดึง UserID ของตัวเองจาก Token 
	userID := c.GetString("user_id") 
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งานในระบบ"})
		return
	}

	// รับค่า Year (ค.ศ.) จาก Query String
	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year == 0 {
		year = time.Now().Year() // ถ้าไม่ส่งมา หรือส่งมามั่ว ให้ดึงปีปัจจุบัน
	}

	res, err := h.repo.GetUserStatistic(userID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการดึงสถิติ: " + err.Error()})
		return
	}

	// ส่งกลับให้ Frontend
	c.JSON(http.StatusOK, res)
}

// 🌟 Get Working Hours Statistic
func (h *StatHandler) GetWorkingHoursStatistic(c *gin.Context) {
	// ดึง UserID ของคนรีเควส จาก Token
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งานในระบบ"})
		return
	}

	res, err := h.repo.GetWorkingHoursStatistic(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาด: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// 🌟 Get Statistic Filter Range
func (h *StatHandler) GetStatisticFilterRange(c *gin.Context) {
	// ดึง UserID จาก Token 
	userID := c.GetString("user_id") 
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งานในระบบ"})
		return
	}

	res, err := h.repo.GetStatisticFilterRange(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการดึงข้อมูล: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}