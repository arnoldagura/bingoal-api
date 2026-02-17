package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BoardMember struct {
	ID       uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BoardID  uuid.UUID      `json:"boardId" gorm:"type:uuid;index;not null"`
	UserID   uuid.UUID      `json:"userId" gorm:"type:uuid;index;not null"`
	Role     string         `json:"role" gorm:"not null;default:'member'"` // owner, member
	JoinedAt time.Time      `json:"joinedAt"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations (for preloading)
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (bm *BoardMember) BeforeCreate(tx *gorm.DB) error {
	if bm.ID == uuid.Nil {
		bm.ID = uuid.New()
	}
	if bm.JoinedAt.IsZero() {
		bm.JoinedAt = time.Now()
	}
	return nil
}
