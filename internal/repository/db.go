package repository

import (
	"log"
	"time" // 🚩 อย่าลืม import time

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, err
	}

	// 🚩 [NEW] ตั้งค่า Connection Pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxIdleConns: จำนวน Connection ที่เปิดรอไว้ (ว่างงาน)
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns: จำนวน Connection สูงสุดที่เปิดได้พร้อมกัน (ป้องกัน DB น็อค)
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime: อายุของ Connection (เพื่อรีเฟรชใหม่กัน Connection เน่า)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Connected to Database with Connection Pool!")
	return db, nil
}