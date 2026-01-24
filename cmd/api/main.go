package main

import (
    "log"
    "my-app/internal/handler"
    "github.com/gofiber/fiber/v2"
)

func main() {
    app := fiber.New()

    // สร้าง Route: เมื่อ Frontend เรียก GET /api/user จะได้ข้อมูล User กลับไป
    app.Get("/api/user", handler.GetUser)

    log.Fatal(app.Listen(":3000"))
}