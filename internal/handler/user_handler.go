package handler

import (
    "my-app/internal/entity"
    "github.com/gofiber/fiber/v2"
)

// GetUser เป็นฟังก์ชันจำลองการส่งข้อมูล User กลับไป
func GetUser(c *fiber.Ctx) error {
    // สมมติข้อมูลขึ้นมา (Later: ดึงจาก DB)
    mockUser := entity.UserInfo{
        UserID:      "U001",
        FullNameEng: "Kriangdet P.",
        Email:       "test@example.com",
    }
    
    // ส่งข้อมูลกลับเป็น JSON
    return c.JSON(mockUser)
}