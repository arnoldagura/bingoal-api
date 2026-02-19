package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Email          string         `json:"email" gorm:"uniqueIndex;not null"`
	Password       string         `json:"-"`
	AuthProvider   string         `json:"authProvider" gorm:"default:email"`
	Name           string         `json:"name"`
	DisplayName    string         `json:"displayName"`
	AvatarURL      string         `json:"avatarUrl"`
	Bio            string         `json:"bio"`
	DailyStreak    int            `json:"dailyStreak" gorm:"default:0"`
	TotalGems      int            `json:"totalGems" gorm:"default:0"`
	LastActiveDate *time.Time     `json:"lastActiveDate"`
	FCMToken       string         `json:"-" gorm:"column:fcm_token"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	Boards         []Board        `json:"boards,omitempty" gorm:"foreignKey:UserID"`
}


func (u *User) Level() string {
	switch {
	case u.TotalGems >= 2000:
		return "diamond"
	case u.TotalGems >= 500:
		return "gold"
	case u.TotalGems >= 100:
		return "silver"
	default:
		return "bronze"
	}
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// Auth DTOs
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"idToken" validate:"required"`
}

type UpdateProfileRequest struct {
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
	Bio         *string `json:"bio"`
	Name        *string `json:"name"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
