package handler

import(
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

    // 🚩 3. แก้จุดนี้: ส่ง userInfo.Role เข้าไปสร้าง JWT
    myToken, err := utils.GenerateToken(userInfo.Role)
    if err != nil {
        c.JSON(500, gin.H{"error": "Internal server error: token generation failed"})
        return
    }

    // 🚩 4. แก้จุดส่งกลับ: แนบ userInfo ไปทั้งก้อนเลย
    c.JSON(200, gin.H{
        "access_token": myToken,
        "user":         userInfo,    // ส่ง ID, Name, Role ไปในก้อนเดียว
        "picture":      googleUser.Picture,
    })
}

func (h *UserHandler) GetUserInfo(c *gin.Context) {
    // 1. รับ ID จาก Path Parameter (เช่น /api/user/:id)
    id := c.Param("id")
    if id == "" {
        c.JSON(400, gin.H{"error": "User ID is required"})
        return
    }

    // 2. เรียกใช้ Repo ที่เพิ่งทำไป
    userInfo, err := h.repo.GetUserInfo(id)
    if err != nil {
        // หากไม่พบข้อมูลหรือเกิด Error ใน DB
        c.JSON(404, gin.H{"error": err.Error()})
        return
    }

    // 3. ส่งข้อมูลกลับ (ข้อมูลชุดใหญ่ที่มีทั้ง Phone, Gender, etc.)
    c.JSON(200, userInfo)
}

func (h *UserHandler) Logout(c *gin.Context) {
   c.JSON(200,gin.H{"status" : "logout ok"})
}


