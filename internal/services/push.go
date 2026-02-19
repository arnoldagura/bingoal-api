package services

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// PushService handles sending push notifications via Firebase Cloud Messaging
type PushService struct {
	client *messaging.Client
}

// Global push service instance
var Push *PushService

// InitPush initializes the Firebase push notification service.
// Returns nil gracefully if no service account is configured (dev mode).
func InitPush(serviceAccountPath string) error {
	if serviceAccountPath == "" {
		log.Println("FCM: No service account configured, push notifications disabled")
		Push = &PushService{client: nil}
		return nil
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		log.Printf("FCM: Failed to initialize Firebase app: %v", err)
		Push = &PushService{client: nil}
		return nil
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("FCM: Failed to get messaging client: %v", err)
		Push = &PushService{client: nil}
		return nil
	}

	Push = &PushService{client: client}
	log.Println("FCM: Push notifications enabled")
	return nil
}

// SendToUser sends a push notification to a user by their ID.
// No-op if push is not configured or user has no FCM token.
func (p *PushService) SendToUser(userID uuid.UUID, title, body string, data map[string]string) {
	if p.client == nil {
		return
	}

	var user models.User
	if err := database.DB.Select("fcm_token").First(&user, userID).Error; err != nil {
		return
	}

	if user.FCMToken == "" {
		return
	}

	msg := &messaging.Message{
		Token: user.FCMToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	if data != nil {
		msg.Data = data
	}

	_, err := p.client.Send(context.Background(), msg)
	if err != nil {
		log.Printf("FCM: Failed to send to user %s: %v", userID, err)
	}
}
