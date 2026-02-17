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

// ... (โค้ดเดิม) ...

// --- Attendance Time Handlers ---

// GetAttendanceTime: GET /system/config/attendance_time/get
func (h *ConfigHandler) GetAttendanceTime(c *gin.Context) {
	data, err := h.repo.GetConfig("attendance_time")
	if err != nil {
		// ถ้ายังไม่มี Config ให้ส่งค่า Default กลับไป (ตามโจทย์)
		defaultConfig := entity.ConfigAttendanceTime{
			AutoCheckout:      true,
			CutoffTime:        entity.TimePair{Hour: 0, Minute: 0},
			CheckInTime:       entity.TimePair{Hour: 8, Minute: 30},
			CheckOutTime:      entity.TimePair{Hour: 16, Minute: 30},
			CheckInLeaveTime:  entity.TimePair{Hour: 13, Minute: 0},
			CheckOutLeaveTime: entity.TimePair{Hour: 12, Minute: 0},
		}
		c.JSON(http.StatusOK, defaultConfig)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateAttendanceTime: PUT /system/config/attendance_time/update
func (h *ConfigHandler) UpdateAttendanceTime(c *gin.Context) {
	var req entity.ConfigAttendanceTime

	// 1. รับค่าและ Validate เบื้องต้น (ตาม Struct Binding)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Logic Validation เพิ่มเติม (ถ้ามี) 
    // เช่น เวลาเข้างานต้องมาก่อนเวลาเลิกงาน
	if req.CheckInTime.Hour > req.CheckOutTime.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Check-in time must be before check-out time"})
		return
	}

	// 3. บันทึกลง DB
	if err := h.repo.SaveConfig("attendance_time", req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance time updated successfully", "data": req})
}

func (h *ConfigHandler) GetAttendanceRequest(c *gin.Context) {
	data, err := h.repo.GetConfig("attendance_request")
	if err != nil {
		// Default Config (ตั้งค่าตามความเหมาะสม)
		defaultConfig := entity.ConfigAttendanceRequest{
			RequestNeedSignature:  true,
			ApproveNeedSignature:  true,
			SpecifyApprovalReason: true,
			SpecifyRemark:         false,
			RequiredRemark:        true,
			EvidenceFile:          true,
			RequiredEvidenceFile:  true,
		}
		c.JSON(http.StatusOK, defaultConfig)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateAttendanceRequest: PUT /system/config/attendance_request/update
func (h *ConfigHandler) UpdateAttendanceRequest(c *gin.Context) {
	var req entity.ConfigAttendanceRequest

	// 1. รับค่า JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Logic Validation (ถ้ามี)
    // เช่น: ถ้าบังคับแนบไฟล์ (RequiredEvidenceFile=true) 
    // ตัว EvidenceFile ก็ต้องเปิดใช้งาน (True) ด้วย
    if req.RequiredEvidenceFile && !req.EvidenceFile {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Evidence file must be enabled if it is required"})
        return
    }

	// 3. บันทึกลง DB
	if err := h.repo.SaveConfig("attendance_request", req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance request config updated", "data": req})
}

// GetLeaveConfig: GET /system/config/leave/get
func (h *ConfigHandler) GetLeaveConfig(c *gin.Context) {
	data, err := h.repo.GetConfig("leave_config")
	if err != nil {
		// ถ้ายังไม่มีข้อมูล ให้ส่ง Default สำหรับประเภทหลักๆ ไปก่อน
		defaultSettings := entity.LeaveSettings{
			RequestNeedSignature: false,
			ApproveNeedSignature: true,
			AllowRetroactive:     false,
			SpecifyRemark:        true,
			RequiredRemark:       true,
			EvidenceFile:         true,
			RequiredEvidenceFile: true,
		}

		// สร้าง Default Map
		defaultConfig := entity.ConfigLeave{
			"sick":      defaultSettings,
			"personal":  defaultSettings,
			"vacation":  defaultSettings,
			"maternity": defaultSettings,
			"paternity": defaultSettings,
			"parental":  defaultSettings,
		}
		c.JSON(http.StatusOK, defaultConfig)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateLeaveConfig: PUT /system/config/leave/update
func (h *ConfigHandler) UpdateLeaveConfig(c *gin.Context) {
	// เนื่องจาก JSON เป็น Map key=string เราใช้ map[string]entity.LeaveSettings รับค่าได้เลย
	var req entity.ConfigLeave

	// 1. รับค่า JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. บันทึกลง DB (เก็บทั้ง Map เป็น JSON ก้อนเดียว)
	if err := h.repo.SaveConfig("leave_config", req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave config updated", "data": req})
}