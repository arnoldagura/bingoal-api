package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Goal struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BoardID     uuid.UUID      `json:"boardId" gorm:"type:uuid;index;not null"`
	Position    int            `json:"position" gorm:"not null"` // 0-24 for 5x5 grid
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	ImageURL    *string        `json:"imageUrl"`
	IsCompleted bool           `json:"isCompleted" gorm:"default:false"`
	CompletedAt *time.Time     `json:"completedAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (g *Goal) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

// Goal DTOs
type UpdateGoalRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"imageUrl"`
	IsCompleted *bool   `json:"isCompleted"`
}
