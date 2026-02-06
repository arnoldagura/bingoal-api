package handlers

import (
	"time"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetBoards(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var boards []models.Board
	if err := database.DB.Where("user_id = ?", userID).
		Preload("Goals").
		Order("created_at DESC").
		Find(&boards).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch boards",
		})
	}

	// Convert to summaries
	summaries := make([]models.BoardSummary, len(boards))
	for i, board := range boards {
		goalCount := 0
		completedCount := 0
		for _, goal := range board.Goals {
			if goal.Title != nil && *goal.Title != "" && goal.Position != 12 { // Exclude grace cell
				goalCount++
				if goal.IsCompleted {
					completedCount++
				}
			}
		}
		summaries[i] = models.BoardSummary{
			ID:             board.ID,
			Title:          board.Title,
			Year:           board.Year,
			IsDefault:      board.IsDefault,
			GoalCount:      goalCount,
			CompletedCount: completedCount,
		}
	}

	return c.JSON(summaries)
}

func GetBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).
		Preload("Goals").
		First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	return c.JSON(board)
}

func CreateBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.CreateBoardRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	year := req.Year
	if year == 0 {
		year = time.Now().Year()
	}

	// Check if this is the first board (make it default)
	var count int64
	database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Count(&count)

	board := models.Board{
		UserID:    userID,
		Title:     req.Title,
		Year:      year,
		IsDefault: count == 0,
	}

	if err := database.DB.Create(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create board",
		})
	}

	// Create 25 empty goals for the board
	goals := make([]models.Goal, 25)
	for i := 0; i < 25; i++ {
		goals[i] = models.Goal{
			BoardID:  board.ID,
			Position: i,
		}
	}

	if err := database.DB.Create(&goals).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create goals",
		})
	}

	// Reload with goals
	database.DB.Preload("Goals").First(&board, board.ID)

	return c.Status(fiber.StatusCreated).JSON(board)
}

func UpdateBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	var req models.UpdateBoardRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title != nil {
		board.Title = *req.Title
	}

	if req.IsDefault != nil && *req.IsDefault {
		// Unset other boards as default
		database.DB.Model(&models.Board{}).
			Where("user_id = ? AND id != ?", userID, boardID).
			Update("is_default", false)
		board.IsDefault = true
	}

	if err := database.DB.Save(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update board",
		})
	}

	return c.JSON(board)
}

func DeleteBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	// Check board count
	var count int64
	database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Count(&count)
	if count <= 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete the last board",
		})
	}

	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	wasDefault := board.IsDefault

	// Delete goals first
	database.DB.Where("board_id = ?", boardID).Delete(&models.Goal{})

	// Delete board
	if err := database.DB.Delete(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete board",
		})
	}

	// If deleted board was default, set another as default
	if wasDefault {
		var newDefault models.Board
		if err := database.DB.Where("user_id = ?", userID).First(&newDefault).Error; err == nil {
			newDefault.IsDefault = true
			database.DB.Save(&newDefault)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
