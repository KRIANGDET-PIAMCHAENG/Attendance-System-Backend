package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"my-app/internal/repository" // อย่าลืมเช็คชื่อโปรเจกต์ของลูกพี่
)

type StatHandler struct {
	repo *repository.PersonnelRepo // 🌟 เปลี่ยนมาใช้ PersonnelRepo แทน
}

func NewStatHandler(repo *repository.PersonnelRepo) *StatHandler {
	return &StatHandler{repo: repo}
}

// 🌟 Get User Statistic (ใช้ Repo ของ Manager)
func (h *StatHandler) GetUserStatistic(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งาน"})
		return
	}

	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year == 0 {
		year = time.Now().Year()
	}

	// 🌟 โยน userID ไปทั้งช่อง managerID และ personnelID
	res, err := h.repo.GetPersonnelStatistic(userID, userID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// 🌟 Get Working Hours Statistic (ใช้ Repo ของ Manager)
func (h *StatHandler) GetWorkingHoursStatistic(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งาน"})
		return
	}

	// 🌟 โยน userID ไป 2 ช่องเช่นกัน
	res, err := h.repo.GetWorkingHoursStatistic(userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// 🌟 Get Statistic Filter Range (ใช้ Repo ของ Manager)
func (h *StatHandler) GetStatisticFilterRange(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งาน"})
		return
	}

	// 🌟 ใช้ฟังก์ชัน GetManagerStatFilterRange ของ PersonnelRepo ได้เลย
	res, err := h.repo.GetManagerStatFilterRange(userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}