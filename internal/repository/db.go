package repository

import(
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(dsn string) (*gorm.DB, error) {
    // เพิ่ม .Debug() เพื่อให้ GORM พ่น SQL ออกมาใน Terminal
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    return db.Debug(), nil 
}
