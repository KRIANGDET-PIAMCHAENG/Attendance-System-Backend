package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"my-app/internal/handler"
	"my-app/internal/middleware"
	"my-app/internal/repository"
	"os"
	"time"
)

func main() {
	// 1. Load .env file (อ่านค่าจากไฟล์ .env เข้าระบบ)
	// ถ้าหาไฟล์ไม่เจอ จะพ่น log เตือน (แต่ไม่ error พัง) เผื่อรันบน Docker ที่ set env ไว้แล้ว
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Warning: No .env file found, using system environment variables")
	}

	// 2. Database Connection
	// อ่านค่า DB_DSN จาก Environment Variable
	dsn := os.Getenv("DB_DSN")

	// เช็คหน่อยว่าลืมใส่ค่ามาหรือเปล่า
	if dsn == "" {
		log.Fatal("❌ Error: DB_DSN is not set in .env")
	}

	db, err := repository.NewDB(dsn)
	if err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	// 2. Setup Layers (Dependency Injection)
	userRepo := repository.NewUserRepo(db)
	userHdl := handler.NewUserHandler(userRepo)

	configRepo := repository.NewConfigRepo(db)
	configHdl := handler.NewConfigHandler(configRepo)

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

	logout := r.Group("/auth")
	logout.Use(middleware.JWTMiddleware())
	{
		logout.POST("/logout", userHdl.Logout)
	}

	profile := r.Group("/profile")
	profile.Use(middleware.JWTMiddleware())
	{
		profile.GET("/me", userHdl.GetUserInfo)
	}

	api := r.Group("/api")
	api.Use(middleware.JWTMiddleware())
	{
		api.GET("/init", userHdl.InitInfo)

	}

	system := r.Group("/system")
	system.Use(middleware.JWTMiddleware()) // ใช้ Middleware ตรวจสอบสิทธิ์
	{
		userMgmt := system.Group("/user_management")
		{

			userMgmt.GET("/users", userHdl.GetAllUsers)

			userMgmt.GET("/roles", userHdl.GetAllRoles)

			userMgmt.GET("/leave/quotas/:id", userHdl.GetUserLeaveQuotas)

			userMgmt.PUT("/roles/update", userHdl.UpdateRole)

			// 1. Create User
			// POST /system/user_management/create
			userMgmt.POST("/create", userHdl.CreateUserSystem)

			// 2. Update Roles
			// PUT /system/user_management/update_role/:id
			userMgmt.PUT("/update_role/:id", userHdl.UpdateUserRoles)

			// 3. Update Max Leave
			// PUT /system/user_management/update_max_leave/:id
			userMgmt.PUT("/update_max_leave/:id", userHdl.UpdateMaxLeave)

			// แถม: Update User Info (ถ้า Frontend ใช้ Path นี้ด้วย)
			// userMgmt.PUT("/update/:id", userHdl.UpdateUser)

			userMgmt.PUT("/update/:id", userHdl.UpdateUser)

			userMgmt.DELETE("/delete", userHdl.DeleteUser)

		}

		roleMgmt := system.Group("/role_management")
		{
			roleMgmt.GET("/role", userHdl.GetRolesWithSubordinatesHandler)

			roleMgmt.GET("/all-user/:id", userHdl.GetNonSubordinatesHandler)

			roleMgmt.PUT("/update/:id", userHdl.UpdateRoleWithMembersHandler)

			roleMgmt.DELETE("/delete", userHdl.DeleteRoleHandler)

			roleMgmt.POST("/create", userHdl.CreateRoleHandler)
		}

		configGroup := system.Group("/config")
		{
			// Budget Year
			configGroup.GET("/budget_year/get", configHdl.GetBudgetYear)
			configGroup.PUT("/budget_year/update", configHdl.UpdateBudgetYear)

			configGroup.GET("/attendance_time/get", configHdl.GetAttendanceTime)
            configGroup.PUT("/attendance_time/update", configHdl.UpdateAttendanceTime)

			configGroup.GET("/attendance_request/get", configHdl.GetAttendanceRequest)
            configGroup.PUT("/attendance_request/update", configHdl.UpdateAttendanceRequest)

			configGroup.GET("/leave/get", configHdl.GetLeaveConfig)
            configGroup.PUT("/leave/update", configHdl.UpdateLeaveConfig)
		}
	}

	// 4. Start Server
	log.Println("🚀 Server running on http://localhost:3000")
	// แนะนำให้ใส่ 0.0.0.0 เพื่อให้เข้าถึงได้จากภายนอก Docker/Network
	r.Run("0.0.0.0:3000")
}
