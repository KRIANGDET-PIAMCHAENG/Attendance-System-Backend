package handler

import (
	"net/http"
	"my-app/internal/service"
	"encoding/json"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Request struct
type GoogleLoginRequest struct {
	Code string `json:"code"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	// 1. Parse JSON body
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Call Service
	sessionID, err := h.authService.LoginWithGoogle(r.Context(), req.Code)
	if err != nil {
		if err.Error() == "user_not_found" {
			// ตอบ Frontend ว่าหา user ไม่เจอ
			http.Error(w, "User not found within organization", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Login failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Success -> Return Session ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":    "Login successful",
		"session_id": sessionID,
	})
}