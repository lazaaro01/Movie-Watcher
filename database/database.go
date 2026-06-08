package database

import (
	"os"
	"path/filepath"

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

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(
		&models.Cinema{},
		&models.Movie{},
		&models.Session{},
		&models.Monitoring{},
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
