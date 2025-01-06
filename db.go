package main

import (
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open db: %v", err.Error())
	}

	initDB(db)

	return db
}

func initDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get underlying sql.DB: %v", err.Error())
	}

	// may help with db locking in larger queries
	sqlDB.SetConnMaxLifetime(time.Minute * 5)
	db.Exec("PRAGMA busy_timeout=60000;")
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	db.Exec("PRAGMA wal_autocheckpoint = 0;")

	err = db.AutoMigrate(&History{}, &Habit{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err.Error())
	}
}
