package utils

import (
    "github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("R1clvDeLZgp5knHvm0WLkBvqMD51khuRBzw1BTjXjH8=")

// สร้าง Struct สำหรับ Claims
type MyCustomClaims struct {
    UserID string `json:"id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims // ตัวนี้จะช่วยจัดการเรื่องเวลาหมดอายุ (ExpiresAt) ให้อัตโนมัติ
}

func GenerateToken(role string, id string) (string, error) {
    claims := MyCustomClaims{
        UserID: id,
        Role:   role,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secretKey)
}