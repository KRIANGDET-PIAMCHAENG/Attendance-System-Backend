package main

import (
	"log"
	"time"
	"my-app/internal/handler"
	"my-app/internal/repository"
	// "my-app/internal/middleware" // Uncomment เมื่อเขียน middleware เสร็จ
	
	"github.com/gin-contrib/cors" // เพิ่มตัวนี้เข้ามา
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Database Connection
	// อย่าลืมเช็ก Password และ DSN ให้ตรงกับที่ใช้ใน Docker นะครับ
	dsn := "postgres://postgres:admin1234@localhost:5432/postgres?sslmode=disable"
	db, err := repository.NewDB(dsn)
	if err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	// 2. Setup Layers (Dependency Injection)
	userRepo := repository.NewUserRepo(db)
	userHdl := handler.NewUserHandler(userRepo)

	// 3. Initialize Router
	r := gin.Default()

	// --- [NEW] ติดตั้ง CORS Middleware เพื่อให้ Flutter (Web/Mobile) ยิงข้าม Domain ได้ ---
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // อนุญาตทุกที่ (เหมาะสำหรับช่วงพัฒนา)
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // ให้ Browser จำค่า OPTIONS ไว้ 12 ชม.
	}))

	// --- กลุ่ม API ที่ไม่ต้องมี Token (Public) ---
	auth := r.Group("/auth")
	{
		// หน้าด่านรับ Access Token จาก Google เพื่อแลก JWT ของเรา
		auth.POST("/google", userHdl.LoginWithGoogle)
	}

	// --- กลุ่ม API ที่ต้องมี JWT ถึงจะเข้าได้ (Protected) ---
	// เมื่อคุณพร้อมใช้ Middleware ให้ Uncomment ส่วนนี้ออกครับ
	/*
	api := r.Group("/api")
	api.Use(middleware.JWTMiddleware()) // ตัวกรองบัตรผ่าน (ต้องไปสร้างไฟล์นี้ใน internal/middleware)
	{
		// ตัวอย่าง API ที่ต้องใช้ Token เข้าถึง
		api.GET("/user-role/:email", userHdl.GetRole) 
	}
	*/

	// 4. Start Server
	log.Println("🚀 Server running on http://localhost:3000")
	// แนะนำให้ใส่ 0.0.0.0 เพื่อให้เข้าถึงได้จากภายนอก Docker/Network
	r.Run("0.0.0.0:3000")
}
