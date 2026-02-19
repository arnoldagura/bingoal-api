package handlers

import (
	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AddComment adds a comment to a goal on a shared board
func AddComment(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	goalID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid goal ID",
		})
	}

	var req models.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Comment text is required",
		})
	}

	// Check goal exists and get board for access check
	var goal models.Goal
	if err := database.DB.First(&goal, goalID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Goal not found",
		})
	}

	if !isBoardMember(goal.BoardID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have access to this board",
		})
	}

	comment := models.Comment{
		GoalID: goalID,
		UserID: userID,
		Text:   req.Text,
	}
	if err := database.DB.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add comment",
		})
	}

	// Preload user for response
	database.DB.Preload("User").First(&comment, comment.ID)

	// Log activity
	LogActivity(goal.BoardID, userID, "comment_added", &goalID, nil)

	// Notify goal owner if different from commenter
	if goal.CompletedBy != nil && *goal.CompletedBy != userID {
		var commenter models.User
		database.DB.First(&commenter, userID)
		name := commenter.DisplayName
		if name == "" {
			name = commenter.Name
		}
		goalTitle := ""
		if goal.Title != nil {
			goalTitle = *goal.Title
		}
		CreateNotification(*goal.CompletedBy, "comment_received",
			"New comment!",
			name+" commented on \""+goalTitle+"\"",
			map[string]interface{}{"boardId": goal.BoardID.String(), "goalId": goalID.String()},
		)
	}

	// Broadcast via WebSocket
	WS.Broadcast(goal.BoardID, userID, WSEvent{
		Type:    EventCommentAdded,
		BoardID: goal.BoardID.String(),
		UserID:  userID.String(),
		Data: map[string]interface{}{
			"goalId":    goalID.String(),
			"commentId": comment.ID.String(),
		},
	})

	return c.Status(fiber.StatusCreated).JSON(comment)
}

// GetGoalComments returns all comments for a goal
func GetGoalComments(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	goalID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid goal ID",
		})
	}

	var goal models.Goal
	if err := database.DB.First(&goal, goalID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Goal not found",
		})
	}

	if !isBoardMember(goal.BoardID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have access to this board",
		})
	}

	var comments []models.Comment
	database.DB.Where("goal_id = ?", goalID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments)

	return c.JSON(comments)
}

// DeleteComment deletes a comment (only by the comment author)
func DeleteComment(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	commentID, err := uuid.Parse(c.Params("commentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid comment ID",
		})
	}

	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Comment not found",
		})
	}

	if comment.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You can only delete your own comments",
		})
	}

	var goal models.Goal
	database.DB.First(&goal, comment.GoalID)

	database.DB.Delete(&comment)

	// Broadcast via WebSocket
	WS.Broadcast(goal.BoardID, userID, WSEvent{
		Type:    EventCommentDeleted,
		BoardID: goal.BoardID.String(),
		UserID:  userID.String(),
		Data: map[string]interface{}{
			"goalId":    comment.GoalID.String(),
			"commentId": commentID.String(),
		},
	})

	return c.JSON(fiber.Map{"success": true})
}
