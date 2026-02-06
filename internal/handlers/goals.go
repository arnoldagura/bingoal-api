package handlers

import (
	"strconv"
	"time"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func UpdateGoal(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("boardId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	position, err := strconv.Atoi(c.Params("position"))
	if err != nil || position < 0 || position > 24 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position (must be 0-24)",
		})
	}

	// Grace cell (position 12) cannot be edited
	if position == 12 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot edit the grace cell",
		})
	}

	// Verify board ownership
	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Find or create goal
	var goal models.Goal
	result := database.DB.Where("board_id = ? AND position = ?", boardID, position).First(&goal)
	if result.Error != nil {
		goal = models.Goal{
			BoardID:  boardID,
			Position: position,
		}
	}

	var req models.UpdateGoalRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title != nil {
		goal.Title = req.Title
	}
	if req.Description != nil {
		goal.Description = req.Description
	}
	if req.ImageURL != nil {
		goal.ImageURL = req.ImageURL
	}
	if req.IsCompleted != nil {
		goal.IsCompleted = *req.IsCompleted
		if *req.IsCompleted {
			now := time.Now()
			goal.CompletedAt = &now
		} else {
			goal.CompletedAt = nil
		}
	}

	if err := database.DB.Save(&goal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update goal",
		})
	}

	return c.JSON(goal)
}

func ToggleGoalCompletion(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("boardId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	position, err := strconv.Atoi(c.Params("position"))
	if err != nil || position < 0 || position > 24 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position (must be 0-24)",
		})
	}

	// Grace cell (position 12) cannot be toggled
	if position == 12 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot toggle the grace cell",
		})
	}

	// Verify board ownership
	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	var goal models.Goal
	if err := database.DB.Where("board_id = ? AND position = ?", boardID, position).First(&goal).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Goal not found",
		})
	}

	goal.IsCompleted = !goal.IsCompleted
	if goal.IsCompleted {
		now := time.Now()
		goal.CompletedAt = &now
	} else {
		goal.CompletedAt = nil
	}

	if err := database.DB.Save(&goal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to toggle goal",
		})
	}

	return c.JSON(goal)
}
