package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MiniGoalMember tracks per-member mini-goal completion on shared boards.
type MiniGoalMember struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	MiniGoalID uuid.UUID      `json:"miniGoalId" gorm:"type:uuid;not null;uniqueIndex:idx_minigoal_user"`
	UserID     uuid.UUID      `json:"userId" gorm:"type:uuid;not null;uniqueIndex:idx_minigoal_user"`
	IsComplete bool           `json:"isComplete" gorm:"default:false"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (mgm *MiniGoalMember) BeforeCreate(tx *gorm.DB) error {
	if mgm.ID == uuid.Nil {
		mgm.ID = uuid.New()
	}
	return nil
}
