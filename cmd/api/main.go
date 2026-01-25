package main

import (
	"log"
	"net/http"
	//"os"

	// เปลี่ยน path ตรงนี้ให้ตรงกับชื่อ module ใน go.mod ของคุณ
	"my-app/internal/entity"
	"my-app/internal/handler"
	"my-app/internal/repository"
	"my-app/internal/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Setup Database Connection
	// dsn (Data Source Name) สำหรับต่อ Postgres ใน Docker
	// host=localhost เพราะเรา run go บนเครื่อง host แต่ db อยู่ใน docker (mapping port 5432 ออกมาแล้ว)
	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// (Optional) Auto Migrate: ให้ Gorm สร้างตารางใน DB ให้ตรงกับ Struct อัตโนมัติ
	// เหมาะสำหรับช่วง Dev จะได้ไม่ต้องไป Create Table เอง
	err = db.AutoMigrate(&entity.User{}, &entity.Role{}, &entity.UserRole{}, &entity.UserInfo{})
	if err != nil {
		log.Println("Migration failed:", err)
	}
	log.Println("Database connected and migrated successfully.")

	// 2. Dependency Injection (เชื่อมต่อเลเยอร์ต่างๆ เข้าด้วยกัน)
	// เริ่มจากล่างขึ้นบน: Repo -> Service -> Handler
	
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	// 3. Setup Router (กำหนดเส้นทาง URL)
	mux := http.NewServeMux()

	// Route สำหรับ Login
	// Frontend จะยิงมาที่ http://localhost:8080/auth/google/login
	mux.HandleFunc("/auth/google/login", authHandler.GoogleLogin)

	// 4. Setup CORS (Cross-Origin Resource Sharing)
	// จำเป็นมาก! เพราะ Frontend (เช่น port 3000) กับ Backend (port 8080) อยู่คนละพอร์ต
	// ถ้าไม่มี Browser จะบล็อกไม่ให้ Frontend ส่ง Code มา
	handlerWithCORS := enableCORS(mux)

	// 5. Start Server
	port := ":8080"
	log.Println("Server is starting on port", port)
	if err := http.ListenAndServe(port, handlerWithCORS); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// ฟังก์ชันแถม: จัดการ CORS แบบง่ายๆ เพื่อให้ Frontend เรียก API ได้
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// อนุญาตให้เข้าถึงได้จากทุกที่ (หรือจะระบุเจาะจงเป็น http://localhost:3000 ก็ได้)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// ถ้าเป็น Preflight request (Browser เช็คก่อนส่งจริง) ให้ตอบ OK กลับไปเลย
		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}