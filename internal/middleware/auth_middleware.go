package middleware

import (
	"net/http"
	"strings"
	"my-app/pkg/utils" 

	"github.com/gin-gonic/gin"
)

var IsTestMode = false

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if IsTestMode {
			c.Set("user_id", "1250101587399")
			// 🚩 ใส่ Role จำลองให้ด้วย ไม่งั้น Test Mode จะเข้าหน้า Admin ไม่ได้
			c.Set("user_roles", []string{"admin"}) 
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort() 
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 🚩 เรียกใช้ ValidateToken จาก utils แทนการ Parse เอง (ลดความผิดพลาดเรื่อง Key)
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 🚩 ฝากข้อมูลลง Context ด้วย Key ที่ถูกต้อง ("user_roles" มี s)
		c.Set("user_id", claims.UserID)
		c.Set("user_roles", claims.Roles) // ใช้ Roles (Array)

		c.Next()
	}
}