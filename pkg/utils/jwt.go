package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Struct สำหรับ Payload ใน Token
type MyCustomClaims struct {
	UserID string   `json:"id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// 🚩 ฟังก์ชันช่วยอ่าน Secret Key (เพื่อให้อ่านค่าหลังจากโหลด .env แล้ว)
func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// กรณีลืมตั้งใน .env ให้ใช้ค่า Default นี้ (ค่าเดียวกับที่คุณส่งมา)
		return []byte("R1clvDeLZgp5knHvm0WLkBvqMD51khuRBzw1BTjXjH8=")
	}
	return []byte(secret)
}

// GenerateToken สร้าง Token ใหม่
func GenerateToken(roles []string, id string) (string, error) {
	claims := MyCustomClaims{
		UserID: id,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "my-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 🚩 เรียก getSecretKey() แทนตัวแปร Global
	return token.SignedString(getSecretKey())
}

// ValidateToken ตรวจสอบ Token
func ValidateToken(tokenString string) (*MyCustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 🚩 เรียก getSecretKey() ตรงนี้ด้วย
		return getSecretKey(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyCustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}