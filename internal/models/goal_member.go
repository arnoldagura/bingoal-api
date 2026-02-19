package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GoalMember tracks per-member goal status on shared boards.
type GoalMember struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID      uuid.UUID      `json:"goalId" gorm:"type:uuid;not null;uniqueIndex:idx_goal_user"`
	UserID      uuid.UUID      `json:"userId" gorm:"type:uuid;not null;uniqueIndex:idx_goal_user"`
	Status      string         `json:"status" gorm:"not null;default:'not_started'"`
	IsCompleted bool           `json:"isCompleted" gorm:"default:false"`
	Progress    int            `json:"progress" gorm:"default:0"`
	CompletedAt *time.Time     `json:"completedAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (gm *GoalMember) BeforeCreate(tx *gorm.DB) error {
	if gm.ID == uuid.Nil {
		gm.ID = uuid.New()
	}
	return nil
}
