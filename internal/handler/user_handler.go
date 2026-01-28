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

    // 🚩 แก้จุดนี้: เปลี่ยนจาก VerifyGoogleToken เป็น VerifyGoogleAccessToken
    googleUser, err := utils.VerifyGoogleAccessToken(input.Token) 
    if err != nil {
        // ถ้า Verify ไม่ผ่าน (Token เน่า/ปลอม) จะส่ง 401 กลับไป
        c.JSON(401, gin.H{"error": "Invalid Google Access Token"})
        return
    }

    // 3. ดึง Role จาก DB ด้วย email ที่ได้มาจาก Google
    role, err := h.repo.GetRoleByEmail(googleUser.Email)
    if err != nil || role == "" {
        c.JSON(401, gin.H{"error": "User not registered in our system"})
        return
    }

    // 4. ออก JWT ของเราเอง (ส่ง role เข้าไป)
    myToken, err := utils.GenerateToken(role)
    if err != nil {
        c.JSON(500, gin.H{"error": "Internal server error: token generation failed"})
        return
    }

    // 5. ส่งกลับไปให้ Client
    c.JSON(200, gin.H{"access_token": myToken})
}