package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoalMemory struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID       uuid.UUID      `json:"goalId" gorm:"type:uuid;not null;index"`
	ImageURL     string         `json:"imageUrl" gorm:"not null"`
	Label        string         `json:"label" gorm:"default:''"`
	IsBoardImage bool           `json:"isBoardImage" gorm:"default:false"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (m *GoalMemory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type CreateGoalMemoryRequest struct {
	ImageURL string `json:"imageUrl"`
	Label    string `json:"label"`
}

type UpdateGoalMemoryRequest struct {
	Label        *string `json:"label"`
	IsBoardImage *bool   `json:"isBoardImage"`
}
