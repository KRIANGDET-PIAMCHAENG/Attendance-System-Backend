package handler

import (
	//"encoding/json"
	"my-app/internal/entity"
	"my-app/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	repo *repository.ConfigRepo
}

func NewConfigHandler(repo *repository.ConfigRepo) *ConfigHandler {
	return &ConfigHandler{repo: repo}
}

// --- Budget Year Handlers ---

// GetBudgetYear: GET /system/config/budget_year/get
func (h *ConfigHandler) GetBudgetYear(c *gin.Context) {
	// 1. ดึงข้อมูลจาก DB
	data, err := h.repo.GetConfig("budget_year")
	if err != nil {
		// ถ้าไม่เจอ (เพิ่งเริ่มระบบ) ให้ส่งค่า Default กลับไป
		defaultConfig := entity.ConfigBudgetYear{Day: 1, Month: 10} // ปกติงบเริ่ม ต.ค.
		c.JSON(http.StatusOK, defaultConfig)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateBudgetYear: PUT /system/config/budget_year/update
func (h *ConfigHandler) UpdateBudgetYear(c *gin.Context) {
	var req entity.ConfigBudgetYear

	// 1. รับค่า JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Validate (เช่น เดือนต้อง 1-12, วันต้อง 1-31)
	if req.Month < 1 || req.Month > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month (1-12)"})
		return
	}
	if req.Day < 1 || req.Day > 31 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day (1-31)"})
		return
	}

	// 3. บันทึกลง DB
	if err := h.repo.SaveConfig("budget_year", req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget year updated successfully", "data": req})
}