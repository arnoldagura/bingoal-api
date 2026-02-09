package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Board struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID      `json:"userId" gorm:"type:uuid;index;not null"`
	Title     string         `json:"title" gorm:"not null"`
	Year      int            `json:"year" gorm:"not null"`
	GridSize  int            `json:"gridSize" gorm:"not null;default:5"`
	Category         *string        `json:"category" gorm:"default:null"`
	GraceSquareTitle *string        `json:"graceSquareTitle" gorm:"default:null"`
	IsDefault        bool           `json:"isDefault" gorm:"default:false"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Goals     []Goal         `json:"goals,omitempty" gorm:"foreignKey:BoardID"`
}

func (b *Board) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// Board DTOs
type CreateBoardRequest struct {
	Title            string  `json:"title" validate:"required"`
	Year             int     `json:"year"`
	GridSize         int     `json:"gridSize"`
	Category         *string `json:"category"`
	GraceSquareTitle *string `json:"graceSquareTitle"`
}

type UpdateBoardRequest struct {
	Title     *string `json:"title"`
	IsDefault *bool   `json:"isDefault"`
}

type BoardSummary struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Year           int       `json:"year"`
	GridSize       int       `json:"gridSize"`
	Category       *string   `json:"category"`
	IsDefault      bool      `json:"isDefault"`
	GoalCount      int       `json:"goalCount"`
	CompletedCount int       `json:"completedCount"`
}
