package utils

import (
	"context"
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