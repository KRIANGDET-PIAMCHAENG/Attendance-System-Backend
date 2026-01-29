package middleware

import (
	"net/http"
	"strings"
	//"my-app/pkg/utils" // ตรวจสอบ path ให้ตรงกับโปรเจกต์คุณ

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. ดึง Header "Authorization" ออกมา
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort() // สั่งหยุด ไม่ให้ไปต่อที่ Handler
			return
		}

		// 2. แยกคำว่า "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. ตรวจสอบ Token (Parse และ Verify Signature)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบว่าใช้ Signing Method เดียวกันไหม (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			// คืนค่า Secret Key ที่เราใช้ (ต้องตรงกับใน pkg/utils/jwt.go)
			return []byte("R1clvDeLZgp5knHvm0WLkBvqMD51khuRBzw1BTjXjH8="), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 4. (Optional) ดึงข้อมูลจาก Payload ออกมาฝากไว้ใน Context
		// เพื่อให้ Handler ตัวต่อไปรู้ว่าคนนี้ Role อะไร
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("role", claims["role"])
		}

		c.Next() // บัตรผ่าน! ไปทำงานที่ Handler ต่อได้
	}
}