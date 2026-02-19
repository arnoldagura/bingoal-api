package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Comment struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	GoalID    uuid.UUID      `json:"goalId" gorm:"type:uuid;index;not null"`
	UserID    uuid.UUID      `json:"userId" gorm:"type:uuid;not null"`
	Text      string         `json:"text" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type CreateCommentRequest struct {
	Text string `json:"text"`
}
