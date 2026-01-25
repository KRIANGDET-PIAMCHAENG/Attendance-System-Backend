package service

import (
	"context"
	"errors"
	//"my-app/internal/entity"
	"my-app/internal/repository"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

type AuthService interface {
	LoginWithGoogle(ctx context.Context, authCode string) (string, error)
}

type authService struct {
	userRepo repository.UserRepository
	// Config ของ OAuth2 (เดี๋ยวไป setup ตอน init)
	oauthConf *oauth2.Config
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{
		userRepo: userRepo,
		oauthConf: &oauth2.Config{
			ClientID:     "PLACEHOLDER_CLIENT_ID",     // เดี๋ยวมาแก้ตอนได้ของจริง
			ClientSecret: "PLACEHOLDER_CLIENT_SECRET", // เดี๋ยวมาแก้ตอนได้ของจริง
			RedirectURL:  "http://localhost:3000",     // URL ของ Frontend หรือ Callback
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (s *authService) LoginWithGoogle(ctx context.Context, authCode string) (string, error) {
	// 1. เอา Code ไปแลก Token
	token, err := s.oauthConf.Exchange(ctx, authCode)
	if err != nil {
		return "", errors.New("failed to exchange code: " + err.Error())
	}

	// 2. ดึง id_token ออกมา (Google จะส่งมาใน extra field)
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return "", errors.New("no id_token field in oauth2 token")
	}

	// 3. Verify id_token ว่าถูกต้องจริงไหม (ป้องกันคนปลอม token มา)
	// ต้องใส่ ClientID ตัวเดิมลงไปเพื่อเช็ค audience
	payload, err := idtoken.Validate(ctx, rawIDToken, s.oauthConf.ClientID)
	if err != nil {
		return "", errors.New("invalid id_token: " + err.Error())
	}

	// 4. ได้ Email มาแล้ว
	email := payload.Claims["email"].(string)

	// 5. เช็คใน DB ว่ามี User คนนี้ไหม
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// *** Logic ที่คุณต้องการ: ถ้าไม่เจอ ให้บอกว่าไม่เจอ (ไม่ต้องสร้างใหม่) ***
			return "", errors.New("user_not_found")
		}
		return "", err
	}

	// 6. ถ้าเจอ User -> สร้าง Session ID (จำลองการสร้าง Session)
	// ในของจริงคุณอาจจะ Gen UUID หรือ JWT Token ส่งกลับไป
	sessionID := "mock-session-id-" + user.UserID 
	
	return sessionID, nil
}