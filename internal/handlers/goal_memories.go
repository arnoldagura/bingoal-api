package handlers

import (
	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func CreateGoalMemory(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	var req models.CreateGoalMemoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if req.ImageURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "imageUrl is required",
		})
	}

	// Check if this will be the first memory for this goal
	var count int64
	database.DB.Model(&models.GoalMemory{}).Where("goal_id = ?", goal.ID).Count(&count)
	isBoardImage := count == 0

	memory := models.GoalMemory{
		GoalID:       goal.ID,
		ImageURL:     req.ImageURL,
		Label:        req.Label,
		IsBoardImage: isBoardImage,
	}

	if err := database.DB.Create(&memory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create memory",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(memory)
}

func ListGoalMemories(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	var memories []models.GoalMemory
	database.DB.Where("goal_id = ?", goal.ID).Order("created_at ASC").Find(&memories)

	return c.JSON(memories)
}

func UpdateGoalMemory(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	memoryID, err := uuid.Parse(c.Params("memoryId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid memory ID",
		})
	}

	var memory models.GoalMemory
	if err := database.DB.Where("id = ? AND goal_id = ?", memoryID, goal.ID).First(&memory).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Memory not found",
		})
	}

	var req models.UpdateGoalMemoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Label != nil {
		memory.Label = *req.Label
	}

	if req.IsBoardImage != nil && *req.IsBoardImage {
		// Clear board image flag on all other memories for this goal first
		database.DB.Model(&models.GoalMemory{}).
			Where("goal_id = ? AND id != ?", goal.ID, memoryID).
			Update("is_board_image", false)
		memory.IsBoardImage = true
	}

	if err := database.DB.Save(&memory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update memory",
		})
	}

	return c.JSON(memory)
}

func DeleteGoalMemory(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	memoryID, err := uuid.Parse(c.Params("memoryId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid memory ID",
		})
	}

	var memory models.GoalMemory
	if err := database.DB.Where("id = ? AND goal_id = ?", memoryID, goal.ID).First(&memory).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Memory not found",
		})
	}

	wasBoardImage := memory.IsBoardImage

	if err := database.DB.Delete(&memory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete memory",
		})
	}

	// If the deleted memory was the board image, promote the oldest remaining memory
	if wasBoardImage {
		var next models.GoalMemory
		if err := database.DB.Where("goal_id = ?", goal.ID).Order("created_at ASC").First(&next).Error; err == nil {
			database.DB.Model(&next).Update("is_board_image", true)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
