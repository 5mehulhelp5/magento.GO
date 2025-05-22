package config

import (
	"fmt"
	"os"
	"log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func NewDB() (*gorm.DB, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		user := os.Getenv("MYSQL_USER")
		pass := os.Getenv("MYSQL_PASS")
		host := os.Getenv("MYSQL_HOST")
		port := os.Getenv("MYSQL_PORT")
		db := os.Getenv("MYSQL_DB")
		if port == "" { port = "3306" }
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", user, pass, host, port, db)
	}

	logMode := logger.Info
	if os.Getenv("GORM_LOG") == "off" {
		logMode = logger.Silent
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // Use log.Logger for Printf support
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      logMode,     // Log level
			Colorful:      true,        // Enable color
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		PrepareStmt: true, // Enable prepared statements
	})
	if err != nil {
		return nil, err
	}

	// Get generic database object
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    // Configure connection pool
    sqlDB.SetMaxOpenConns(25)       // Maximum open connections
    sqlDB.SetMaxIdleConns(25)       // Maximum idle connections
    sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Maximum connection lifetime
    sqlDB.SetConnMaxIdleTime(2 * time.Minute)  // Maximum idle time
	
	return db, nil
} 