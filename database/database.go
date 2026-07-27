package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/opencode/movie-watcher/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	gormLogger := logger.New(log.New(os.Stderr, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(
		&models.Cinema{},
		&models.Movie{},
		&models.Session{},
		&models.Monitoring{},
		&models.PriceHistory{},
	)
	if err != nil {
		return err
	}

	DB = db
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
