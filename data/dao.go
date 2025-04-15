package data

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	gorm.DB
}

func OpenDatabase(conn string) *DB {
	db, err := gorm.Open(sqlite.Open(conn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{}, &NotificationChannel{})
	return &DB{*db}
}
