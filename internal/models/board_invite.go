package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BoardInvite struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BoardID    uuid.UUID      `json:"boardId" gorm:"type:uuid;index;not null"`
	InviterID  uuid.UUID      `json:"inviterId" gorm:"type:uuid;not null"`
	InviteCode string         `json:"inviteCode" gorm:"uniqueIndex;not null"`
	ExpiresAt  *time.Time     `json:"expiresAt"`
	MaxUses    int            `json:"maxUses" gorm:"default:0"` // 0 = unlimited
	UsedCount  int            `json:"usedCount" gorm:"default:0"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (bi *BoardInvite) BeforeCreate(tx *gorm.DB) error {
	if bi.ID == uuid.Nil {
		bi.ID = uuid.New()
	}
	if bi.InviteCode == "" {
		bi.InviteCode = generateInviteCode()
	}
	return nil
}

// IsValid checks if the invite is still usable
func (bi *BoardInvite) IsValid() bool {
	if bi.ExpiresAt != nil && time.Now().After(*bi.ExpiresAt) {
		return false
	}
	if bi.MaxUses > 0 && bi.UsedCount >= bi.MaxUses {
		return false
	}
	return true
}

func generateInviteCode() string {
	b := make([]byte, 6) // 12 hex chars
	rand.Read(b)
	return hex.EncodeToString(b)
}

type CreateInviteRequest struct {
	MaxUses   int `json:"maxUses"`   // 0 = unlimited
	ExpiresIn int `json:"expiresIn"` // hours, 0 = never
}
