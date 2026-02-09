package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MiniGoal struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID     uuid.UUID      `json:"goalId" gorm:"type:uuid;index;not null"`
	Title      string         `json:"title" gorm:"not null"`
	Percentage int            `json:"percentage" gorm:"not null"`
	IsComplete bool           `json:"isComplete" gorm:"default:false"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (m *MiniGoal) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// MiniGoal DTOs
type CreateMiniGoalRequest struct {
	Title      string `json:"title" validate:"required"`
	Percentage int    `json:"percentage" validate:"required"`
}

type UpdateMiniGoalRequest struct {
	Title      *string `json:"title"`
	Percentage *int    `json:"percentage"`
}
