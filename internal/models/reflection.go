package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reflection struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID           uuid.UUID      `json:"goalId" gorm:"type:uuid;uniqueIndex;not null"`
	Obstacles        *string        `json:"obstacles"`
	Victories        *string        `json:"victories"`
	Notes            *string        `json:"notes"`
	ReflectionPrompt *string        `json:"reflectionPrompt"`
	ReflectionAnswer *string        `json:"reflectionAnswer"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (r *Reflection) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// Reflection DTOs
type UpsertReflectionRequest struct {
	Obstacles        *string `json:"obstacles"`
	Victories        *string `json:"victories"`
	Notes            *string `json:"notes"`
	ReflectionAnswer *string `json:"reflectionAnswer"`
}

// Random reflection prompts
var ReflectionPrompts = []string{
	"What surprised you most about achieving this goal?",
	"What would you do differently if you started over?",
	"Who helped you along the way?",
	"What skill did you develop while working on this?",
	"How has completing this goal changed your perspective?",
	"What was the hardest moment, and how did you push through?",
	"What are you most proud of in this journey?",
	"How will you build on this accomplishment?",
}
