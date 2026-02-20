package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Goal struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BoardID     uuid.UUID      `json:"boardId" gorm:"type:uuid;index;not null"`
	Position    int            `json:"position" gorm:"not null"`
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Icon        *string        `json:"icon"`
	ImageURL    *string        `json:"imageUrl"`
	AssignedTo   *uuid.UUID     `json:"assignedTo" gorm:"type:uuid"`
	CompletedBy  *uuid.UUID     `json:"completedBy" gorm:"type:uuid"`
	Mood         *string        `json:"mood"`                                          // sage, terracotta, slate, sunrise
	Status       string         `json:"status" gorm:"not null;default:'not_started'"` // not_started, in_progress, completed
	IsCompleted  bool           `json:"isCompleted" gorm:"default:false"`
	IsGraceSquare bool          `json:"isGraceSquare" gorm:"default:false"`
	Progress      int            `json:"progress" gorm:"default:0"`
	CompletedAt   *time.Time     `json:"completedAt"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	MiniGoals     []MiniGoal     `json:"miniGoals,omitempty" gorm:"foreignKey:GoalID"`
	Reflection    *Reflection    `json:"reflection,omitempty" gorm:"foreignKey:GoalID"`

	// Transient field — populated by API for shared boards, not stored in DB
	CompletedByCount int `json:"completedByCount,omitempty" gorm:"-"`
}

func (g *Goal) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}


type UpdateGoalRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Icon        *string    `json:"icon"`
	ImageURL    *string    `json:"imageUrl"`
	Mood        *string    `json:"mood"`
	IsCompleted *bool      `json:"isCompleted"`
	AssignedTo  *uuid.UUID `json:"assignedTo"`
}
