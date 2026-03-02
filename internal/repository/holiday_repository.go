package repository

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// 1. สร้าง Struct สำหรับ HolidayRepo โดยเฉพาะ
type HolidayRepo struct {
	db *gorm.DB
}

func NewHolidayRepo(db *gorm.DB) *HolidayRepo {
	return &HolidayRepo{db: db}
}

// 2. Struct รับข้อมูลจาก BOT API
type BOTHolidayResponse struct {
	Result struct {
		Data []struct {
			Date                   string `json:"Date"`
			HolidayDescriptionThai string `json:"HolidayDescriptionThai"`
		} `json:"data"`
	} `json:"result"`
}

type HolidayData struct {
	Date        time.Time
	Description string
	Year        int
}

// 3. ฟังก์ชันบันทึกข้อมูลลง DB
func (r *HolidayRepo) SyncHolidays(holidays []HolidayData) error {
	for _, h := range holidays {
		sql := `
			INSERT INTO company_holidays (holiday_date, description, year) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (holiday_date) DO NOTHING
		`
		if err := r.db.Exec(sql, h.Date, h.Description, h.Year).Error; err != nil {
			log.Println("Error inserting holiday:", err)
		}
	}
	return nil
}

// 4. ฟังก์ชันหลักสำหรับดึง API และบันทึก
func (r *HolidayRepo) FetchAndSyncHolidays() error {
	log.Println("[CRON] กำลังดึงข้อมูลวันหยุดจาก BOT API...")

	url := "https://gateway.api.bot.or.th/financial-institutions-holidays/"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")
	
	// 🔑 ใส่ API Key ของคุณตรงนี้นะครับ!
	req.Header.Add("Authorization", "eyJvcmciOiI2NzM1NzgwZWM4YzFlYjAwMDEyYTM3NzEiLCJpZCI6ImMwZDYyYWEyNzAwMTRlYmI5MjE4NjFlMzA1OTUxNjlmIiwiaCI6Im11cm11cjEyOCJ9")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Println("[CRON] ดึงข้อมูลไม่สำเร็จ:", err)
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var botRes BOTHolidayResponse
	if err := json.Unmarshal(body, &botRes); err != nil {
		log.Println("[CRON] แปลง JSON ไม่สำเร็จ:", err)
		return err
	}

	var holidaysToSave []HolidayData
	for _, item := range botRes.Result.Data {
		parsedDate, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			continue 
		}
		holidaysToSave = append(holidaysToSave, HolidayData{
			Date:        parsedDate,
			Description: item.HolidayDescriptionThai,
			Year:        parsedDate.Year(),
		})
	}

	// เรียกฟังก์ชันเซฟลง DB
	err = r.SyncHolidays(holidaysToSave)
	if err == nil {
		log.Printf("[CRON] อัปเดตวันหยุดสำเร็จ! จัดการไป %d วัน\n", len(holidaysToSave))
	}
	return err
}