package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reaction struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID    uuid.UUID      `json:"goalId" gorm:"type:uuid;index;not null"`
	UserID    uuid.UUID      `json:"userId" gorm:"type:uuid;not null"`
	Type      string         `json:"type" gorm:"not null"` // fire, heart, clap, star
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (r *Reaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type CreateReactionRequest struct {
	Type string `json:"type" validate:"required"` // fire, heart, clap, star
}
