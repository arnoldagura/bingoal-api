package database

import (
	"strings"

	"github.com/arnold/bingoals-api/internal/config"
	"github.com/arnold/bingoals-api/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	var dialector gorm.Dialector

	// Use PostgreSQL if URL starts with postgres, otherwise SQLite
	if strings.HasPrefix(cfg.DatabaseURL, "postgres") {
		dialector = postgres.Open(cfg.DatabaseURL)
	} else {
		dialector = sqlite.Open(cfg.DatabaseURL)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	DB = db
	return nil
}

func Migrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Board{},
		&models.Goal{},
		&models.MiniGoal{},
		&models.Reflection{},
	)
}
