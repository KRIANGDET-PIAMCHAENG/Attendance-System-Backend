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
    // 1. รับ Google Token จาก Client
    // (สมมติว่า Client ส่ง JSON {"token": "google_access_token"})
    var input struct { Token string `json:"token"` }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // 2. [SKIP] ตรงนี้คุณต้องเขียนฟังก์ชันไป Verify กับ Google API 
    // เพื่อให้ได้ email มา (สมมติว่าได้ email มาแล้ว)
    googleUser, err := utils.VerifyGoogleToken(input.Token)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid Google Token"})
        return
    }

    // 3. ใช้ฟังก์ชันเดิมที่คุณมี ดึง Role จาก DB
    role, err := h.repo.GetRoleByEmail(googleUser.Email)
    if err != nil || role == "" {
        c.JSON(401, gin.H{"error": "User not registered in our system"})
        return
    }

    // 4. ออก JWT ของเราเอง!
    myToken, err := utils.GenerateToken(role)
    if err != nil {
        c.JSON(500, gin.H{"error": "Could not generate token"})
        return
    }

    // 5. ส่งกลับไปให้ Client
    c.JSON(200, gin.H{"access_token": myToken})
}