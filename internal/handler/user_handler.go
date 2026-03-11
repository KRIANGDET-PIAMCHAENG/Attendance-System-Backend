package handler

import(
    "log"
	"net/http"
	"my-app/internal/repository" 
	"github.com/gin-gonic/gin"
	"my-app/pkg/utils"
	"fmt"
	"time"
	"sort"
	"os"
	"path/filepath"
	"github.com/google/uuid"
)

type UserHandler struct {
	repo *repository.UserRepo 
}

func NewUserHandler(repo *repository.UserRepo) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) LoginWithGoogle(c *gin.Context) {
	// รับ Token จาก Frontend
	var input struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format"})
		return
	}

	// 1. Verify Google Token
	googleUser, err := utils.VerifyGoogleAccessToken(input.Token)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid Google Access Token", "details": err.Error()})
		return
	}

	// 2. ดึงข้อมูล User จาก Database ด้วย Email
	// (ใช้ฟังก์ชัน GetUserInfoByEmail ที่มีอยู่แล้วใน Repo)
	userInfo, err := h.repo.GetUserInfoByEmail(googleUser.Email)
	if err != nil {
		// ถ้า Error แปลว่าหาไม่เจอ หรือ DB มีปัญหา
		c.JSON(401, gin.H{"error": "User not registered in our system"})
		return
	}

	// 3. ดึง Role ทั้งหมดของ User คนนี้ (สำคัญมาก! เพื่อเอาไปใส่ใน JWT)
	// สมมติว่าใน Repo มีฟังก์ชัน GetUserRoles(userID) ที่คืนค่า []string
	// *หมายเหตุ: ถ้ายังไม่มีฟังก์ชันนี้ใน Repo ให้ไปเพิ่มก่อนนะครับ (ดูโค้ดตัวอย่างด้านล่าง)
	roles, err := h.repo.GetUserRoles(userInfo.UserID) 
	if err != nil {
		// กรณีดึง Role ไม่ได้ ให้ถือว่า User ไม่มี Role หรือ Log error ไว้
		// แต่เพื่อให้ Login ผ่านได้ อาจจะให้ roles เป็น empty slice ไปก่อน
		log.Printf("Failed to get user roles: %v", err)
		roles = []string{} 
	}

	// 4. Update รูปโปรไฟล์ (ถ้ามี)
	if err := h.repo.UpdatePicture(googleUser.Email, googleUser.Picture); err != nil {
		log.Printf("Failed to update picture: %v", err)
		// ไม่ return error เพื่อให้ Login ต่อได้
	}

	// 5. Generate JWT Token
	// 🚩 ส่ง roles (ที่เป็น []string) เข้าไปแทน string ตัวเดียว
	myToken, err := utils.GenerateToken(userInfo.Role, userInfo.UserID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Internal server error: token generation failed"})
        return
    }

	// 6. ส่ง Response กลับไปให้ Frontend
	// เราอัปเดต picture ใน DB แล้ว ดังนั้นค่าใน userInfo.Picture (ถ้า Get มาใหม่) ก็น่าจะอัปเดตแล้ว
	// หรือจะส่ง googleUser.Picture กลับไปให้เลยก็ได้เพื่อความสดใหม่
	c.JSON(200, gin.H{
		"access_token": myToken,
		"user": gin.H{
			"user_id": userInfo.UserID,
			"name":    userInfo.Name,
			"email":   userInfo.Email,
			"roles":   roles,              // ส่ง Role ทั้งหมดกลับไปด้วย
			"picture": googleUser.Picture, // ใช้รูปจาก Google ล่าสุด
		},
	})
}




func (h *UserHandler) Logout(c *gin.Context) {
   c.JSON(200,gin.H{"status" : "logout ok"})
}

func (h *UserHandler) GetUserInfo(c *gin.Context) {
    // 🚩 ดึง user_id ที่ Middleware ฝากไว้ (แกะมาจาก Token)
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(500, gin.H{"error": "User ID not found in context"})
        return
    }

    // เรียกใช้ Repo ด้วย ID ที่ได้จาก Token โดยตรง
    // ป้องกันการที่ User แอบเปลี่ยน ID ใน URL (BOLA Attack Prevention)
    userInfo, err := h.repo.GetUserInfo(userID.(string))
    if err != nil {
        c.JSON(404, gin.H{"error": "Profile not found"})
        return
    }

    c.JSON(200, userInfo)
}

func (h *UserHandler) InitInfo(c *gin.Context) {
    // 1. ดึง user_id จาก Middleware (Context)
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "Unauthorized: No user ID found"})
        return
    }

    // 2. เรียก Repo เพื่อดึงข้อมูลตั้งต้น
    initData, err := h.repo.GetInitInfo(userID.(string))
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch initialization data"})
        return
    }

    // 3. ส่งข้อมูลกลับไปให้ Flutter
    c.JSON(200, initData)
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	// เรียก Repository
	users, err := h.repo.GetAllUsers()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	// ส่งกลับตามโครงสร้าง { "data": [ ... ] }
	c.JSON(200, gin.H{
		"data": users,
	})
}

func (h *UserHandler) GetAllRoles(c *gin.Context) {
	roles, err := h.repo.GetAllRoles()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch roles", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"data": roles,
	})
}

func (h *UserHandler) GetUserLeaveQuotas(c *gin.Context) {
    // 🚩 แก้ไข: รับ ID จาก URL Parameter แทน (เช่น /api/leave/quotas/1250...)
    targetUserID := c.Param("id")

    // (Optional) คุณอาจจะอยากเช็คตรงนี้เพิ่มว่า คนที่เรียก (c.Get("user_id")) เป็น Admin จริงไหม
    // แต่ถ้าเชื่อใจ Middleware ก็ข้ามไปก่อนได้ครับ

    // เรียก Repository ด้วย ID ที่ส่งมา
    quotas, err := h.repo.GetLeaveQuotas(targetUserID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch quotas"})
        return
    }

    // แปลงข้อมูลเป็น JSON Map เหมือนเดิม
    responseMap := make(map[string]float64)
    for _, q := range quotas {
        responseMap[q.TypeKey] = q.DaysAllowed
    }

    c.JSON(200, responseMap)
}

// ... (code เดิม)

func (h *UserHandler) UpdateUser(c *gin.Context) {
	// 1. รับ ID จาก URL (เช่น /api/users/55555...)
	id := c.Param("id")

	var req repository.UpdateUserRequest

	// 2. Bind JSON (เฉพาะ Body)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	// 3. ส่ง ID และ Struct ไป Repo
	err := h.repo.UpdateUserInfo(id, req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update user", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "User updated successfully",
	})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	var req repository.UpdateRoleRequest

	// 1. แปลง JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	// 2. เรียก Repo
	err := h.repo.UpdateRole(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update role", "details": err.Error()})
		return
	}

	// 3. ตอบกลับ
	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Role updated successfully",
		"data":    req,
	})
}

// ... (code เดิม)

// 1. Handler Create User System
func (h *UserHandler) CreateUserSystem(c *gin.Context) {
	var req repository.CreateUserFullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON", "details": err.Error()})
		return
	}

	if err := h.repo.CreateUserFull(req); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}
	c.JSON(201, gin.H{"status": "success", "message": "User created successfully"})
}

func (h *UserHandler) UpdateUserRoles(c *gin.Context) {
    id := c.Param("id")
    var req struct {
        Roles []string `json:"roles"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid JSON", "details": err.Error()})
        return
    }

    // 🚩 ส่ง req.Roles (ที่เป็น []string) ไปเลย
    if err := h.repo.UpdateUserRoles(id, req.Roles); err != nil {
        c.JSON(500, gin.H{"error": "Failed to update roles", "details": err.Error()})
        return
    }
    c.JSON(200, gin.H{"status": "success"})
}
// 3. Handler Update Max Leave
func (h *UserHandler) UpdateMaxLeave(c *gin.Context) {
	id := c.Param("id") // รับ ID จาก URL
	var req repository.MaxLeavePart
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON", "details": err.Error()})
		return
	}

	if err := h.repo.UpdateUserMaxLeave(id, req); err != nil {
		c.JSON(500, gin.H{"error": "Failed to update leave quotas", "details": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// backend/internal/handler/user_handler.go

func (h *UserHandler) DeleteUser(c *gin.Context) {
	// สร้าง Struct สำหรับรับ JSON { "id": "..." }
	var req struct {
		ID string `json:"id" binding:"required"`
	}

	// Bind JSON จาก Body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid JSON format or missing 'id'",
			"details": err.Error(),
		})
		return
	}

	// เรียก Repo เพื่อลบ โดยใช้ req.ID ที่แกะได้
	if err := h.repo.DeleteUser(req.ID); err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to delete user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "User deleted successfully",
	})
}

func (h *UserHandler) GetRolesWithSubordinatesHandler(c *gin.Context) {
	// เรียก Repo
	data, err := h.repo.GetRolesWithSubordinates()
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to fetch roles and subordinates",
			"details": err.Error(),
		})
		return
	}

	// ส่งกลับเป็น JSON ตาม format ที่ Frontend ขอมาเป๊ะๆ
	// { "roles": [ ... ] }
	c.JSON(200, gin.H{
		"roles": data,
	})
}

// GetNonSubordinatesHandler ส่งรายชื่อพนักงานที่ยังไม่ได้เป็นลูกน้องของ Role นั้นกลับไป
func (h *UserHandler) GetNonSubordinatesHandler(c *gin.Context) {
	roleID := c.Param("id") // รับ {role-id} จาก URL

	data, err := h.repo.GetNonSubordinatesByRole(roleID)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to fetch potential members",
			"details": err.Error(),
		})
		return
	}

	// ส่งกลับตาม Format ที่เพื่อนขอเป๊ะๆ
	c.JSON(200, gin.H{
		"members": data,
	})
}

// UpdateRoleWithMembersHandler รับ Request มาอัปเดต Role
func (h *UserHandler) UpdateRoleWithMembersHandler(c *gin.Context) {
	var req repository.UpdateRoleFullRequest

	// 1. Bind JSON เข้า Struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// 2. (Optional) รับ ID จาก URL มาทับใน Body เพื่อความชัวร์
	idFromURL := c.Param("id")
	if idFromURL != "" {
		req.ID = idFromURL
	}

	// 3. เรียก Repository
	err := h.repo.UpdateRoleWithMembers(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update role", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Role updated successfully"})
}


// DeleteRoleRequest struct สำหรับรับค่า ID จาก Body
type DeleteRoleRequest struct {
	ID string `json:"id" binding:"required"`
}

// DeleteRoleHandler รับ Request ลบ Role
func (h *UserHandler) DeleteRoleHandler(c *gin.Context) {
	var req DeleteRoleRequest

	// รับค่า ID จาก Body (เพราะ Frontend ส่งมาเป็น data: {'id': ...})
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// เรียก Repo เพื่อทำการลบ
	err := h.repo.DeleteRole(req.ID)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to delete role",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "Role and its relationships deleted successfully",
		"deleted_id": req.ID,
	})
}

// CreateRoleHandler: POST /system/role/create
func (h *UserHandler) CreateRoleHandler(c *gin.Context) {
	var req repository.CreateRoleRequest

	// 1. รับ JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. เรียก Repository
	if err := h.repo.CreateRole(req); err != nil {
		fmt.Println("Bind Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role created successfully", "id": req.ID})
}


// GetAllMembersHandler: GET /system/user_management/members
func (h *UserHandler) GetAllMembersHandler(c *gin.Context) {
	members, err := h.repo.GetAllMembers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
		return
	}

	// ส่งกลับในรูปแบบ { "members": [...] } ตามที่ขอ
	c.JSON(http.StatusOK, gin.H{
		"members": members,
	})
}

func (h *UserHandler) RecordAttendanceHandler(c *gin.Context) {
	var req repository.RecordAttendanceRequest

	// Bind JSON ตัวใหม่
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid Data", "details": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(string)

	if err := h.repo.RecordAttendance(userID, req); err != nil {
		c.JSON(500, gin.H{"error": "Failed to record", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Success", "timestamp": req.Timestamp})
}

// ฟังก์ชันช่วยแปลง Weekday เป็นภาษาไทย
func getThaiDOW(d time.Weekday) string {
	days := []string{"อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"}
	return days[d]
}

func (h *UserHandler) GetMyAttendanceHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// 1. ดึงข้อมูลจากตาราง attendance (ข้อมูลการสแกนนิ้ว)
	records, err := h.repo.GetAttendanceHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 2. ดึงข้อมูลจากตาราง leave_requests (ข้อมูลการลางาน)
	leaves, _ := h.repo.GetApprovedLeavesForHistory(userID)

	// 3. เตรียม Struct สำหรับส่งเป็น JSON
	type ResponseItem struct {
		Date        string  `json:"date"`
		Dow         string  `json:"dow"`
		CheckIn     *string `json:"checkIn"`
		CheckOut    *string `json:"checkOut"`
		LeavePeriod string  `json:"leavePeriod"`
	}

	// 4. สร้าง Map เพื่อรวมข้อมูล (ใช้ Date เป็น Key)
	historyMap := make(map[string]ResponseItem)

	// 4.1 เอาข้อมูลการเข้างาน (Attendance) ยัดลง Map เป็นตัวตั้งต้น
	for _, r := range records {
		dateStr := r.Date.Format("2006-01-02")

		var checkInStr *string
		if r.CheckIn != nil && len(*r.CheckIn) >= 5 {
			ci := (*r.CheckIn)[:5]
			checkInStr = &ci
		}

		var checkOutStr *string
		if r.CheckOut != nil && len(*r.CheckOut) >= 5 {
			co := (*r.CheckOut)[:5]
			checkOutStr = &co
		}

		historyMap[dateStr] = ResponseItem{
			Date:        dateStr,
			Dow:         getThaiDOW(r.Date.Weekday()),
			CheckIn:     checkInStr,
			CheckOut:    checkOutStr,
			LeavePeriod: "NONE", // ตั้งเป็น NONE ไว้ก่อน
		}
	}

	// 4.2 เอาข้อมูลการลา (Leaves) มาผสมทับลงไป
	for _, l := range leaves {
		// วนลูปตั้งแต่วันที่เริ่มลา จนถึง วันที่สิ้นสุดการลา
		for d := l.DateFrom; !d.After(l.DateTo); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			
			// 🌟 คาดเดาประเภทการลาจากเวลา
			period := "FULL_DAY" 
			if l.DateFrom.Format("2006-01-02") == l.DateTo.Format("2006-01-02") {
				if l.DateTo.Hour() <= 12 {
					period = "MORNING"
				} else if l.DateFrom.Hour() >= 13 {
					period = "AFTERNOON"
				}
			}

			// 🌟 สำคัญ: เช็คว่าวันนั้น "มีการสแกนนิ้ว" (อยู่ใน Map) หรือไม่
			if item, exists := historyMap[dateStr]; exists {
				// ถ้ามีการสแกนนิ้ว ถึงจะเอา LeavePeriod ไปแปะทับ
				item.LeavePeriod = period
				historyMap[dateStr] = item
			}
			// ❌ ลบ else ทิ้ง! ถ้าวันนั้นไม่ได้สแกนนิ้ว (เช่น ลาเต็มวันแล้วอยู่บ้าน) ก็ไม่ต้องโชว์ในประวัติการเข้างาน
		}
	}

	// 5. แปลง Map กลับมาเป็น Slice (Array)
	var result []ResponseItem
	for _, v := range historyMap {
		result = append(result, v)
	}

	// 6. เรียงลำดับวันที่จากใหม่ไปเก่า (DESC)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date > result[j].Date
	})

	// ถ้าไม่มีข้อมูล ให้ส่ง array เปล่าไปแทน null
	if result == nil {
		result = []ResponseItem{}
	}

	c.JSON(http.StatusOK, result)
}

// [NEW] ฟังก์ชันเช็คสถานะ Check-In/Check-Out ของวันนี้ (เวอร์ชันเวลาไทย)
func (h *UserHandler) GetTodayAttendanceStatus(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// 1. ล็อค Timezone เป็นเวลาประเทศไทย (UTC+7)
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		// ถ้าโหลด Timezone ไม่ได้ (บางกรณีใน Docker ที่ไม่มี tzdata) ให้ fallback ใช้เวลาเครื่อง
		loc = time.Local
	}
	
	// ดึงเวลา "ปัจจุบัน" ตามเวลาไทย
	today := time.Now().In(loc)

	// 2. ไปดึงข้อมูลจาก Database
	record, err := h.repo.GetTodayAttendance(userID, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 3. ถ้ายังไม่มีข้อมูลของวันนี้
	if record == nil {
		c.JSON(http.StatusOK, gin.H{
			"checkIn":  nil,
			"checkOut": nil,
		})
		return
	}

	// 4. ตัด string เอาแค่ "HH:mm"
	var checkInStr *string
	if record.CheckIn != nil && len(*record.CheckIn) >= 5 {
		ci := (*record.CheckIn)[:5]
		checkInStr = &ci
	}

	var checkOutStr *string
	if record.CheckOut != nil && len(*record.CheckOut) >= 5 {
		co := (*record.CheckOut)[:5]
		checkOutStr = &co
	}

	c.JSON(http.StatusOK, gin.H{
		"checkIn":  checkInStr,
		"checkOut": checkOutStr,
	})
}

// ==========================================
// 1. GET /signature (ส่งเป็นไฟล์รูปกลับไปเลย)
// ==========================================
func (h *UserHandler) GetSignature(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	path, err := h.repo.GetSignaturePath(userID)
	if err != nil || path == nil || *path == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบลายเซ็น"})
		return
	}

	// เช็คว่าไฟล์มีอยู่จริงบน Server ไหม
	if _, err := os.Stat(*path); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไฟล์ลายเซ็นสูญหาย"})
		return
	}

	// ส่งเป็น Byte Array ให้ Frontend โดยอัตโนมัติ (ตรงสเปค Frontend เป๊ะ)
	c.File(*path)
}

// ==========================================
// 2. PUT /signature/update (รับไฟล์มาเซฟ)
// ==========================================
func (h *UserHandler) UpdateSignature(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	file, err := c.FormFile("signature") // รับคีย์ชื่อ "signature"
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ไม่พบไฟล์ลายเซ็น"})
		return
	}

	// สร้าง Folder ย่อยตามปี/เดือน
	uploadDir := "uploads/signatures/" + time.Now().Format("2006/01")
	os.MkdirAll(uploadDir, os.ModePerm)

	// ตั้งชื่อไฟล์ใหม่ด้วย UUID
	ext := filepath.Ext(file.Filename)
	savePath := filepath.Join(uploadDir, uuid.New().String()+ext)

	// เซฟไฟล์ลง Server
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกไฟล์ไม่สำเร็จ"})
		return
	}

	// อัปเดต Path ลง DB
	if err := h.repo.UpdateSignaturePath(userID, &savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "อัปเดตฐานข้อมูลไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "อัปเดตลายเซ็นเรียบร้อย"})
}

func getWD() string {
    dir, _ := os.Getwd()
    return dir
}

// ==========================================
// 3. DELETE /signature/clear (ลบไฟล์และเคลียร์ DB)
// ==========================================
func (h *UserHandler) ClearSignature(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	path, err := h.repo.GetSignaturePath(userID)
    if err == nil && path != nil && *path != "" {
        // 🔍 ใส่ Log เช็ค Path เต็มๆ อีกรอบ
        log.Printf("Current Working Dir: %s", getWD()) // เดี๋ยวผมให้ฟังก์ชันเสริมด้านล่าง
        log.Printf("Attempting to delete file at: %s", *path)

        errRemove := os.Remove(*path) 
        if errRemove != nil {
            // 🚨 จุดสำคัญ: ดูว่ามันฟ้องว่าอะไร!
            log.Printf("❌ Remove failed: %v", errRemove)
        } else {
            log.Println("✅ File deleted successfully from disk")
        }
    }

	// อัปเดต DB ให้เป็น NULL
	if err := h.repo.UpdateSignaturePath(userID, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เคลียร์ฐานข้อมูลไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ลบลายเซ็นเรียบร้อย"})
}

func (h *AttendanceReqHandler) CheckHoliday(c *gin.Context) {
	// รับวันที่จาก Query เช่น ?date=2026-04-13
	dateStr := c.Query("date") 
	
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุวันที่ (date=YYYY-MM-DD)"})
		return
	}

	holidayName, err := h.repo.CheckHoliday(dateStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถตรวจสอบวันหยุดได้"})
		return
	}

	// ส่งกลับไปให้ Frontend
	c.JSON(http.StatusOK, gin.H{
		"date":         dateStr,
		"holiday_name": holidayName, // ถ้าเป็น nil ฝั่ง Gin จะแปลงเป็น null ให้เลย
	})
}

// [NEW] GET /api/attendance/filter_range
func (h *UserHandler) GetAttendanceFilterRangeHistory(c *gin.Context) {
	// ดึง userID จาก Token ที่ถูกฝากไว้ใน Middleware
	userID := c.MustGet("user_id").(string)

	// เรียก Repository
	res, err := h.repo.GetAttendanceFilterRangeHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch filter range", "details": err.Error()})
		return
	}

	// ส่งกลับในรูปแบบที่ Frontend ต้องการเป๊ะๆ 
	// { "start": "...", "end": "..." }
	c.JSON(http.StatusOK, res)
}