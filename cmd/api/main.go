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

	// อย่าลืม import cron กับ repository ของคุณ
	"github.com/robfig/cron/v3"
	// "your_project/internal/repository"
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

	leaveHdl := handler.NewLeaveHandler(userRepo)

	attendanceReqHdl := handler.NewAttendanceReqHandler(userRepo)

	// 🌟 [NEW] 3.1 ประกาศ HolidayRepo สำหรับจัดการวันหยุด
	holidayRepo := repository.NewHolidayRepo(db)

	personnelRepo := repository.NewPersonnelRepo(db)
	personnelHdl := handler.NewPersonnelHandler(personnelRepo)

	statHdl := handler.NewStatHandler(personnelRepo)

	// ประกาศ Notification
    notificationRepo := repository.NewNotificationRepo(db)
    notificationHdl := handler.NewNotificationHandler(notificationRepo)

	leaveAppvRepo := repository.NewLeaveApprovalRepo(db)
    leaveAppvHdl := handler.NewLeaveApprovalHandler(leaveAppvRepo, notificationRepo)

    attAppvRepo := repository.NewAttendanceApprovalRepo(db)
    attAppvHdl := handler.NewAttendanceApprovalHandler(attAppvRepo, notificationRepo)

	// ==========================================
	// 🌟 [NEW] Setup Cron Job (ทำงานหลังบ้าน)
	// ==========================================
	c := cron.New()

	// ตั้งให้ทำงานทุกๆ วันที่ 1 ของเดือน เวลา 00:00 น. ("นาที ชั่วโมง วัน เดือน วันในสัปดาห์")
	_, cronErr := c.AddFunc("0 0 1 * *", func() {
		err := holidayRepo.FetchAndSyncHolidays()
		if err != nil {
			log.Println("[CRON] Auto Sync Holidays Error:", err)
		}
	})

	if cronErr != nil {
		log.Fatalf("ตั้งค่า Cron Job ไม่สำเร็จ: %v", cronErr)
	}

	c.Start()      // สั่งให้เริ่มเดินนาฬิกา
	defer c.Stop() // ปิด Cron เมื่อ Server ดับ

	// 3. Initialize Router
	r := gin.Default()

	r.Static("/uploads", "./uploads")

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

		attendance := api.Group("/attendance")
		{
			// POST /api/attendance/record
			attendance.POST("/record", userHdl.RecordAttendanceHandler)

			// [NEW] GET /api/attendance/history
			attendance.GET("/history", userHdl.GetMyAttendanceHistory)

			attendance.GET("/today", userHdl.GetTodayAttendanceStatus)

			attendance.GET("/check_holiday", attendanceReqHdl.CheckHoliday)
		}

		leave_request := api.Group("/leave_request")
		{
			leave_request.POST("/create", leaveHdl.CreateLeaveRequest)
			leave_request.GET("/leave_info", leaveHdl.GetLeaveInfo)

			leave_request.PUT("/resend", leaveHdl.ResendLeaveRequest)

			leave_request.DELETE("/cancel", leaveHdl.CancelLeaveRequest)
		}

		leave_status := api.Group("/leave_status")
		{
			leave_status.GET("/pending", leaveHdl.GetPendingLeaves)
			leave_status.GET("/recent", leaveHdl.GetRecentLeaves)

			leave_status.GET("/detail", leaveHdl.GetLeaveDetail)

			leave_status.GET("/filter_range", leaveHdl.GetLeaveFilterRange)

		}

		signature := api.Group("/signature")
		{
			signature.GET("/get", userHdl.GetSignature)
			signature.PUT("/update", userHdl.UpdateSignature)
			signature.DELETE("/clear", userHdl.ClearSignature)
		}

		attendance_req := api.Group("/attendance_request")
		{
			attendance_req.POST("/create", attendanceReqHdl.CreateTimeRequest)
			attendance_req.GET("/pending", attendanceReqHdl.GetPendingRequests)
			attendance_req.GET("/recent", attendanceReqHdl.GetRecentRequests)
			attendance_req.GET("/filter_range", attendanceReqHdl.GetFilterRange)

			attendance_req.GET("/detail", attendanceReqHdl.GetAttendanceDetail)
			attendance_req.DELETE("/delete", attendanceReqHdl.DeleteAttendanceRequest)
			attendance_req.PUT("/resend", attendanceReqHdl.ResendAttendanceRequest)
		}

		notifGroup := api.Group("/notifications")
        {
            notifGroup.GET("", notificationHdl.GetNotifications)
            notifGroup.PATCH("/:id/read", notificationHdl.MarkAsRead)
            notifGroup.PATCH("/read-all", notificationHdl.MarkAllAsRead)
            notifGroup.GET("/unread-count", notificationHdl.GetUnreadCount)
            notifGroup.POST("/send-request", notificationHdl.SendRequestNotification)
            notifGroup.POST("/send-response", notificationHdl.SendResponseNotification)
        }

		leaveAppv := api.Group("/leave-approval")
        {
            leaveAppv.GET("/pending", leaveAppvHdl.GetPendingSummary)
            leaveAppv.GET("/recent", leaveAppvHdl.GetRecent)
            leaveAppv.GET("/filter_range", leaveAppvHdl.GetFilterRange)
            leaveAppv.GET("/user_detail", leaveAppvHdl.GetUserDetail)
            leaveAppv.GET("/request_detail", leaveAppvHdl.GetRequestDetail)
            leaveAppv.PUT("/:id", leaveAppvHdl.ApproveReject)
        }

        // ⏰ ระบบอนุมัติแก้ไขเวลา
        attAppv := api.Group("/attendance-approval")
        {
            attAppv.GET("/pending", attAppvHdl.GetPending)
            attAppv.GET("/recent", attAppvHdl.GetRecent)
            attAppv.GET("/filter_range", attAppvHdl.GetFilterRange)
            attAppv.GET("/detail", attAppvHdl.GetDetail)
            attAppv.PUT("/:id", attAppvHdl.ApproveReject)
        }

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

			userMgmt.GET("/members", userHdl.GetAllMembersHandler)
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

	manager := r.Group("/manager")
	manager.Use(middleware.JWTMiddleware()) // แนะนำให้เปิด Middleware ไว้กันคนนอกแอบดู
	{
		// ✅ เปลี่ยนจาก api.Group เป็น manager.Group ครับ
		personnel := manager.Group("/personnel_info")
		{
			personnel.GET("/leave/pending", personnelHdl.GetPending)
			personnel.GET("/leave/recent", personnelHdl.GetRecent)
			personnel.GET("/leave/filter_range", personnelHdl.GetFilterRange)
			personnel.GET("/leave/detail", personnelHdl.GetDetail)
			personnel.GET("/users", personnelHdl.GetUsers)
			personnel.GET("/permissions", personnelHdl.GetPermissionLevel)

			// 🌟 [NEW] เพิ่มเส้นนี้เข้าไปในกลุ่ม personnel
			personnel.GET("/personnel_data", personnelHdl.GetPersonnelData)

			// 📊 Statistic
			personnel.GET("/statistic/working_hours", personnelHdl.GetWorkingHoursStat)
			personnel.GET("/statistic/filter_range", personnelHdl.GetManagerStatFilterRange)
			personnel.GET("/statistic", personnelHdl.GetStatistic)

			// ⏱️ Attendance Request
			personnel.GET("/attendance_request/pending", personnelHdl.GetAttReqPending)
			personnel.GET("/attendance_request/recent", personnelHdl.GetAttReqRecent)
			personnel.GET("/attendance_request/filter_range", personnelHdl.GetAttReqFilterRange)
			personnel.GET("/attendance_request/detail", personnelHdl.GetAttReqDetail)

			// 📅 Attendance History
			personnel.GET("/attendance/history", personnelHdl.GetAttendanceHistory)
			personnel.GET("/attendance/filter_range", personnelHdl.GetManagerStatFilterRange) // ใช้ฟังก์ชันร่วมกับ Statistic ได้เลย
			


		}



	}

	user_api := r.Group("/user")
	user_api.Use(middleware.JWTMiddleware()) // บังคับแนบ Token
	{
		// 🌟 [NEW] เพิ่มเส้น Statistic
		user_api.GET("/statistic", statHdl.GetUserStatistic)
		user_api.GET("/statistic/working_hours", statHdl.GetWorkingHoursStatistic)
		user_api.GET("/statistic/filter_range", statHdl.GetStatisticFilterRange)
	}

	// 4. Start Server
	log.Println("🚀 Server running on http://localhost:3000")
	// แนะนำให้ใส่ 0.0.0.0 เพื่อให้เข้าถึงได้จากภายนอก Docker/Network
	r.Run("0.0.0.0:3000")
}
