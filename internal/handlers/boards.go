package handlers

import (
	"time"
	"sort"
	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
			if goal.IsGraceSquare {
				continue
			}
			goalCount++
			if goal.IsCompleted {
				completedCount++
			}
		}
		summaries[i] = models.BoardSummary{
			ID:             board.ID,
			Title:          board.Title,
			Year:           board.Year,
			GridSize:       board.GridSize,
			Category:       board.Category,
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
	if err := database.DB.
		Where("id = ? AND user_id = ?", boardID, userID).
		Preload("Goals", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Goals.MiniGoals").
		Preload("Goals.Reflection").
		First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Inject virtual grace square (odd grids only)
	if board.GridSize%2 == 1 {
		centerPos := (board.GridSize * board.GridSize) / 2

		found := false
		for _, g := range board.Goals {
			if g.Position == centerPos {
				found = true
				break
			}
		}

		if !found {
			graceTitle := "Grace"
			if board.GraceSquareTitle != nil && *board.GraceSquareTitle != "" {
				graceTitle = *board.GraceSquareTitle
			}

			grace := models.Goal{
				ID:            uuid.Nil,
				BoardID:       board.ID,
				Position:      centerPos,
				Title:         &graceTitle,
				IsGraceSquare: true,
				IsCompleted:   true,
				Status:        "completed",
				Progress:      100,
			}

			board.Goals = append(board.Goals, grace)

			sort.Slice(board.Goals, func(i, j int) bool {
				return board.Goals[i].Position < board.Goals[j].Position
			})
		}
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

	gridSize := req.GridSize
	if gridSize != 3 && gridSize != 7 {
		gridSize = 5 // Default to 5x5
	}

	var count int64
	database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Count(&count)

	board := models.Board{
		UserID:           userID,
		Title:            req.Title,
		Year:             year,
		GridSize:         gridSize,
		Category:         req.Category,
		GraceSquareTitle: req.GraceSquareTitle,
		IsDefault:        count == 0,
	}

	if err := database.DB.Create(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create board",
		})
	}

	database.DB.Preload("Goals", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&board, board.ID)

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

	var count int64
	database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Count(&count)

	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	wasDefault := board.IsDefault

	database.DB.Where("board_id = ?", boardID).Delete(&models.Goal{})

	if err := database.DB.Delete(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete board",
		})
	}

	if wasDefault {
		var newDefault models.Board
		if err := database.DB.Where("user_id = ?", userID).First(&newDefault).Error; err == nil {
			newDefault.IsDefault = true
			database.DB.Save(&newDefault)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
