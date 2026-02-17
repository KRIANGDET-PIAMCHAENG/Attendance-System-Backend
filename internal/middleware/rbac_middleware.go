package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// RequireAdmin ตรวจสอบว่า User มีสิทธิ์ "admin" หรือไม่ (รองรับ Multiple Roles)
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. ดึง user_roles (เติม s) ที่ฝากไว้ใน Context
		rolesVal, exists := c.Get("user_roles") // 🚩 ต้องใช้ key "user_roles" ให้ตรงกับ auth_middleware
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No roles found"})
			c.Abort()
			return
		}

		// 2. แปลงเป็น []string (ไม่ใช่ string เดียวแล้ว)
		userRoles, ok := rolesVal.([]string)
		if !ok {
			// ถ้าแปลงไม่ได้ แสดงว่า Auth Middleware ส่งมาผิด Type
			c.JSON(500, gin.H{"error": "System error: Invalid roles format"})
			c.Abort()
			return
		}

		// 3. วนลูปเช็คว่ามี "admin" อยู่ในลิสต์ไหม
		isAdmin := false
		for _, r := range userRoles {
			if r == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireHR ก็ต้องแก้ลักษณะเดียวกันครับ
func RequireHR() gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("user_roles")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		
		userRoles, ok := rolesVal.([]string)
		if !ok {
			c.JSON(500, gin.H{"error": "System error"})
			c.Abort()
			return
		}

		isHR := false
		for _, r := range userRoles {
			if r == "hr" || r == "admin" { // Admin เป็น HR ได้
				isHR = true
				break
			}
		}

		if !isHR {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: HR access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}