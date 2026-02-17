package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID      `json:"userId" gorm:"type:uuid;index;not null"`
	Type      string         `json:"type" gorm:"not null"` // board_invite, goal_completed, reaction_received, member_joined
	Title     string         `json:"title" gorm:"not null"`
	Body      string         `json:"body"`
	Read      bool           `json:"read" gorm:"default:false"`
	Metadata  *string        `json:"metadata"` // JSON string for navigation context (boardId, goalId, etc.)
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
