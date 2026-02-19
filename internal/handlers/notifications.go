package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/arnold/bingoals-api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetNotifications returns paginated notifications for the current user
func GetNotifications(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	var notifications []models.Notification
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&notifications)

	var total int64
	database.DB.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&total)

	var unread int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&unread)

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"total":         total,
		"unread":        unread,
		"page":          page,
		"limit":         limit,
	})
}

// MarkNotificationRead marks a single notification as read
func MarkNotificationRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	notifID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid notification ID",
		})
	}

	result := database.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("read", true)

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Notification not found",
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// MarkAllRead marks all notifications as read for the current user
func MarkAllRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true)

	return c.JSON(fiber.Map{"success": true})
}

// RegisterDeviceToken saves the FCM token for push notifications
func RegisterDeviceToken(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", req.Token)

	return c.JSON(fiber.Map{"success": true})
}

// CreateNotification is a helper to create notifications from other handlers
func CreateNotification(userID uuid.UUID, notifType, title, body string, metadata map[string]interface{}) {
	notif := models.Notification{
		UserID: userID,
		Type:   notifType,
		Title:  title,
		Body:   body,
	}

	var pushData map[string]string
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err == nil {
			s := string(data)
			notif.Metadata = &s
		}
		// Convert metadata to string map for push payload
		pushData = make(map[string]string)
		for k, v := range metadata {
			pushData[k] = fmt.Sprintf("%v", v)
		}
		pushData["type"] = notifType
	}

	database.DB.Create(&notif)

	// Send push notification
	if services.Push != nil {
		go services.Push.SendToUser(userID, title, body, pushData)
	}
}

// notifyBoardMembers sends a notification to all members of a board except the actor
func notifyBoardMembers(boardID, excludeUserID uuid.UUID, notifType, title, body string, metadata map[string]interface{}) {
	var members []models.BoardMember
	database.DB.Where("board_id = ? AND user_id != ?", boardID, excludeUserID).Find(&members)

	for _, m := range members {
		CreateNotification(m.UserID, notifType, title, body, metadata)
	}
}
