package middleware

import (
    "github.com/gin-gonic/gin"
)

// MockAuthMiddleware ใช้สำหรับข้ามการเช็ก Token จริงตอน Test
func MockAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 🚩 ยัด ID ของคุณ Kriangdet ที่เรา Insert ไว้ใน DB ลงไปเลย
        c.Set("user_id", "1250101587399")
        c.Set("user_role", "admin")
        
        c.Next()
    }
}