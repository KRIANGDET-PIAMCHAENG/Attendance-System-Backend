package repository

import (
	"encoding/json"
	"my-app/internal/entity"

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