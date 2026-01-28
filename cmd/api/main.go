package main

import (
	"log"
	"my-app/internal/handler"
	"my-app/internal/repository"
	//"my-app/internal/middleware" 
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Database Connection (เหมือนเดิม)
	dsn := "postgres://postgres:admin1234@localhost:5432/postgres?sslmode=disable"
	db, err := repository.NewDB(dsn)
	if err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	// 2. Setup Layers (Dependency Injection)
	userRepo := repository.NewUserRepo(db)
	userHdl  := handler.NewUserHandler(userRepo)

	// 3. Initialize Router
	r := gin.Default()

	// --- กลุ่ม API ที่ไม่ต้องมี Token (Public) ---
	auth := r.Group("/auth")
	{
		// หน้าด่านรับ Access Token จาก Google เพื่อแลก JWT ของเรา
		auth.POST("/google", userHdl.LoginWithGoogle) 
	}

	// --- กลุ่ม API ที่ต้องมี JWT ถึงจะเข้าได้ (Protected) ---
	// เราจะใช้ Middleware ที่เราจะเขียนขึ้นมามา "ดัก" ไว้
	// api := r.Group("/api")
	// api.Use(middleware.JWTMiddleware()) // <--- ตัวกรองบัตรผ่าน
	// {
	// 	// ถ้าไม่มี Token หรือ Token ปลอม จะเข้า function นี้ไม่ได้
	// 	api.GET("/user-role/:email", userHdl.GetRole)
	// }

	// 4. Start Server
	log.Println("🚀 Server running on http://localhost:3000")
	r.Run(":3000")
}