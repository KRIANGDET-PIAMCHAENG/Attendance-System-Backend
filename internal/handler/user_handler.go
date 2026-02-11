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

