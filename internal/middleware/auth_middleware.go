package middleware

import (
	"net/http"
	"strings"
	"my-app/pkg/utils" // ตรวจสอบ path ให้ตรงกับโปรเจกต์คุณ

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
		token, err := jwt.ParseWithClaims(tokenString, &utils.MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return []byte("R1clvDeLZgp5knHvm0WLkBvqMD51khuRBzw1BTjXjH8="), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // 🚩 4. ดึงข้อมูลจาก Payload (Claims) ออกมาฝากไว้ใน Context
        if claims, ok := token.Claims.(*utils.MyCustomClaims); ok && token.Valid {
            // ฝากทั้ง ID และ Role ไว้ใน Context
            c.Set("user_id", claims.UserID)
            c.Set("user_role", claims.Role)
        }

        c.Next()
	}
}