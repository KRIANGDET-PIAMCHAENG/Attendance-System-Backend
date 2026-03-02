package repository

import (
	"encoding/json"
	"my-app/internal/entity"
	"time"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConfigRepo struct {
	db *gorm.DB
}

func NewConfigRepo(db *gorm.DB) *ConfigRepo {
	return &ConfigRepo{db: db}
}

// SaveConfig บันทึกหรืออัปเดตค่า Config (Upsert)
func (r *ConfigRepo) SaveConfig(key string, value interface{}) error {
	// 1. แปลง Struct เป็น Map (JSON)
	jsonBytes, _ := json.Marshal(value)
	var jsonMap map[string]interface{}
	json.Unmarshal(jsonBytes, &jsonMap)

	config := entity.SystemConfig{
		ConfigKey:   key,
		ConfigValue: jsonMap,
	}

	// 2. Upsert: ถ้า Key ซ้ำ ให้ Update Value
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value", "updated_at"}),
	}).Create(&config).Error
}

// GetConfig ดึงค่า Config ตาม Key
func (r *ConfigRepo) GetConfig(key string) (map[string]interface{}, error) {
	var config entity.SystemConfig
	// ค้นหาตาม Key
	err := r.db.First(&config, "config_key = ?", key).Error
	if err != nil {
		return nil, err
	}
	return config.ConfigValue, nil
}


type BudgetConfig struct {
	Day   int `json:"day"`
	Month int `json:"month"`
}

// ฟังก์ชันคำนวณปีงบประมาณ (คืนค่าเป็น Integer เช่น 2026)
func GetBudgetYear(db *gorm.DB, targetDate time.Time) (int, error) {
	var configValue string
	
	// 1. ดึงค่า config จาก DB
	err := db.Table("system_configs").
		Select("config_value").
		Where("config_key = ?", "budget_year").
		Scan(&configValue).Error
		
	if err != nil || configValue == "" {
		// ถ้าไม่มี config ใน DB ให้ default เป็นปีปัจจุบัน
		return targetDate.Year(), nil 
	}

	// 2. แกะ JSON
	var config BudgetConfig
	if err := json.Unmarshal([]byte(configValue), &config); err != nil {
		return targetDate.Year(), err
	}

	year := targetDate.Year()

	// 3. ถ้าระบบตั้งเป็นเริ่ม 1 มกราคม (ไม่คาบเกี่ยวปี) ตอบปีปัจจุบันได้เลย
	if config.Month == 1 && config.Day == 1 {
		return year, nil
	}

	// 4. หาวันที่เริ่มปีงบประมาณของ "ปีปฏิทินปัจจุบัน"
	// สมมติ config คือ 1 ต.ค. -> budgetStart จะเป็น 1 ต.ค. ของปี targetDate
	budgetStart := time.Date(year, time.Month(config.Month), config.Day, 0, 0, 0, 0, targetDate.Location())

	// 5. ตรรกะปัดปีงบประมาณ (อิงตามปีที่สิ้นสุดรอบ)
	if targetDate.Before(budgetStart) {
		// เช่น เช็ค 1 พ.ค. 2026 (ยังไม่ถึง 1 ต.ค. 2026)
		// ถือว่าอยู่ในรอบที่เริ่มมาตั้งแต่ปีที่แล้ว และจะสิ้นสุดปีนี้ -> ปีงบ = 2026
		return year, nil
	} else {
		// เช่น เช็ค 1 พ.ย. 2025 (เลย 1 ต.ค. 2025 มาแล้ว)
		// ถือว่าเข้าสู่รอบใหม่ ที่จะไปสิ้นสุดในปีหน้า -> ปีงบ = 2026
		return year + 1, nil
	}
}