package utils

import (
	"context"
	"encoding/json" // เพิ่มตัวนี้สำหรับ json.NewDecoder
	"net/http"      // เพิ่มตัวนี้สำหรับ http.Get
	"google.golang.org/api/idtoken"
)

// GoogleClientID คือ ID ที่ได้จาก Google Cloud Console
const GoogleClientID = "224641754766-6gt8o9876skh3h4t1p14ooalv3iqc784.apps.googleusercontent.com"

type GoogleUser struct {
	Email string
	Name  string
	Picture string
}

func VerifyGoogleToken(idToken string) (*GoogleUser, error) {
	// 1. ตรวจสอบ Token กับ Google API
	// idtoken.Validate จะเช็ก Signature, Expiration และ Audience (ClientID) ให้เสร็จสรรพ
	payload, err := idtoken.Validate(context.Background(), idToken, GoogleClientID)
	if err != nil {
		return nil, err
	}

	// 2. ดึงข้อมูลจาก Payload (Claims)
	// ข้อมูลในนี้คือสิ่งที่ Google ยืนยันมาแล้วว่าจริง
	user := &GoogleUser{
		Email:   payload.Claims["email"].(string),
		Name:    payload.Claims["name"].(string),
		Picture: payload.Claims["picture"].(string),
	}

	return user, nil
}

func VerifyGoogleAccessToken(accessToken string) (*GoogleUser, error) {
    resp, err := http.Get("https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + accessToken)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 1. แก้ตรงนี้: เพิ่ม Picture ในโครงสร้างรับข้อมูลจาก Google
    var res struct {
        Email   string `json:"email"`
        Name    string `json:"name"`
        Picture string `json:"picture"` // <--- Google ส่งมาใน key ชื่อ "picture"
    }

    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        return nil, err
    }

    // 2. แก้ตรงนี้: ส่งค่า Picture เข้าไปใน GoogleUser struct
    return &GoogleUser{
        Email:   res.Email,
        Name:    res.Name,
        Picture: res.Picture, // <--- ตอนนี้ค่าจะไม่ว่างแล้ว!
    }, nil
}