package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Activity struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BoardID    uuid.UUID      `json:"boardId" gorm:"type:uuid;index;not null"`
	UserID     uuid.UUID      `json:"userId" gorm:"type:uuid;not null"`
	ActionType string         `json:"actionType" gorm:"not null"` // goal_completed, goal_assigned, member_joined, member_left, reaction
	TargetID   *uuid.UUID     `json:"targetId" gorm:"type:uuid"` // goal ID or user ID depending on action
	Metadata   *string        `json:"metadata"`                  // JSON string for extra context
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (a *Activity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
