package handler

import(
    "log"
	"my-app/internal/repository" 
	"github.com/gin-gonic/gin"
	"my-app/pkg/utils"
)

type UserHandler struct {
	repo *repository.UserRepo 
}

func NewUserHandler(repo *repository.UserRepo) *UserHandler {
	return &UserHandler{repo: repo}
}


func (h *UserHandler) LoginWithGoogle(c *gin.Context) {
    var input struct { Token string `json:"token" binding:"required"` }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, gin.H{"error": "Invalid JSON format"})
        return
    }

    // 1. Verify Google Token (เหมือนเดิม)
    googleUser, err := utils.VerifyGoogleAccessToken(input.Token) 
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid Google Access Token"})
        return
    }

    // 🚩 2. เปลี่ยนจุดนี้: จาก GetRoleByEmail เป็น GetUserInfoByEmail
    // เราจะได้รับตัวแปร userInfo ซึ่งเป็นก้อน Struct (ID, Name, Email, Role) มาแทน
    userInfo, err := h.repo.GetUserInfoByEmail(googleUser.Email)
    if err != nil {
        // ถ้าหาไม่เจอ แปลว่าอีเมลนี้ไม่ได้ลงทะเบียนไว้ในระบบเรา
        c.JSON(401, gin.H{"error": "User not registered in our system"})
        return
    }

    // userRole, err := h.repo.AllRole(userInfo.UserID)
    // if err != nil {
    //     // ถ้าหาไม่เจอ แปลว่าอีเมลนี้ไม่ได้ลงทะเบียนไว้ในระบบเรา
    //     c.JSON(401, gin.H{"error": "User not registered in our system"})
    //     return
    // }

    err_update := h.repo.UpdatePicture(googleUser.Email, googleUser.Picture)
    if err_update != nil {
        log.Printf("Failed to update picture: %v", err)
        // ไม่ต้อง return error ก็ได้เพื่อให้ User ยัง Login ต่อได้แม้รูปจะอัปเดตไม่สำเร็จ
    }

    // 🚩 3. แก้จุดนี้: ส่ง userInfo.Role เข้าไปสร้าง JWT
    myToken, err := utils.GenerateToken(userInfo.RoleGen,userInfo.UserID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Internal server error: token generation failed"})
        return
    }

    // 🚩 4. แก้จุดส่งกลับ: แนบ userInfo ไปทั้งก้อนเลย
    c.JSON(200, gin.H{
        "access_token": myToken,
        "user":         userInfo,   
        //"role":         userRole,
        // "picture":      googleUser.Picture,
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
