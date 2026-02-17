package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetBoardActivity returns paginated activity for a board
func GetBoardActivity(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	if !isBoardMember(boardID, userID) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	var activities []models.Activity
	database.DB.Where("board_id = ?", boardID).
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities)

	var total int64
	database.DB.Model(&models.Activity{}).Where("board_id = ?", boardID).Count(&total)

	return c.JSON(fiber.Map{
		"activities": activities,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

// AddReaction adds or toggles a reaction on a goal
func AddReaction(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	goalID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid goal ID",
		})
	}

	var req models.CreateReactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	validTypes := map[string]bool{"fire": true, "heart": true, "clap": true, "star": true}
	if !validTypes[req.Type] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid reaction type. Must be: fire, heart, clap, or star",
		})
	}

	// Check goal exists and get board ID for access check
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

	// Toggle: if same reaction exists, remove it; otherwise add it
	var existing models.Reaction
	if err := database.DB.Where("goal_id = ? AND user_id = ? AND type = ?", goalID, userID, req.Type).First(&existing).Error; err == nil {
		// Remove existing reaction
		database.DB.Delete(&existing)
		return c.JSON(fiber.Map{"removed": true, "type": req.Type})
	}

	// Add new reaction
	reaction := models.Reaction{
		GoalID: goalID,
		UserID: userID,
		Type:   req.Type,
	}
	if err := database.DB.Create(&reaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add reaction",
		})
	}

	// Notify the goal's completer (if different from reactor)
	if goal.CompletedBy != nil && *goal.CompletedBy != userID {
		var reactor models.User
		database.DB.First(&reactor, userID)
		name := reactor.DisplayName
		if name == "" {
			name = reactor.Name
		}
		goalTitle := ""
		if goal.Title != nil {
			goalTitle = *goal.Title
		}
		CreateNotification(*goal.CompletedBy, "reaction_received",
			"New reaction!",
			name+" reacted "+req.Type+" to \""+goalTitle+"\"",
			map[string]interface{}{"boardId": goal.BoardID.String(), "goalId": goalID.String()},
		)
	}

	return c.Status(fiber.StatusCreated).JSON(reaction)
}

// GetReactions returns all reactions for a goal
func GetGoalReactions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	goalID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid goal ID",
		})
	}

	// Check goal exists and verify access
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

	var reactions []models.Reaction
	database.DB.Where("goal_id = ?", goalID).Preload("User").Find(&reactions)

	return c.JSON(reactions)
}

// LogActivity is a helper to create activity entries from other handlers
func LogActivity(boardID, userID uuid.UUID, actionType string, targetID *uuid.UUID, metadata map[string]interface{}) {
	activity := models.Activity{
		BoardID:    boardID,
		UserID:     userID,
		ActionType: actionType,
		TargetID:   targetID,
	}

	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err == nil {
			s := string(data)
			activity.Metadata = &s
		}
	}

	database.DB.Create(&activity)
}
