package handlers

import (
	"time"
	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetBoards(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Get personal boards (owned by user) + shared boards (user is a member)
	var boards []models.Board

	// Find board IDs where user is a member (for shared boards)
	var memberBoardIDs []uuid.UUID
	database.DB.Model(&models.BoardMember{}).
		Where("user_id = ?", userID).
		Pluck("board_id", &memberBoardIDs)

	query := database.DB.
		Preload("Goals").
		Preload("Members.User").
		Order("created_at DESC")

	if len(memberBoardIDs) > 0 {
		// Personal boards by user_id OR shared boards by membership
		query = query.Where("user_id = ? OR id IN ?", userID, memberBoardIDs)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&boards).Error; err != nil {
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
			goalCount++
			if goal.IsCompleted {
				completedCount++
			}
		}

		// Build member info
		var members []models.MemberInfo
		for _, m := range board.Members {
			members = append(members, models.MemberInfo{
				ID:          m.UserID,
				Name:        m.User.Name,
				DisplayName: m.User.DisplayName,
				AvatarURL:   m.User.AvatarURL,
				Role:        m.Role,
			})
		}

		summaries[i] = models.BoardSummary{
			ID:             board.ID,
			Title:          board.Title,
			Year:           board.Year,
			GridSize:       board.GridSize,
			Category:       board.Category,
			BoardType:      board.BoardType,
			MaxMembers:     board.MaxMembers,
			IsDefault:      board.IsDefault,
			GoalCount:      goalCount,
			CompletedCount: completedCount,
			MemberCount:    len(board.Members),
			Members:        members,
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
		Where("id = ?", boardID).
		Preload("Goals", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Goals.MiniGoals").
		Preload("Goals.Reflection").
		Preload("Members.User").
		First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Check access: user must be owner or a member
	if board.UserID != userID {
		var membership models.BoardMember
		if err := database.DB.Where("board_id = ? AND user_id = ?", boardID, userID).First(&membership).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Board not found",
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

	boardType := req.BoardType
	if boardType != "shared" {
		boardType = "personal"
	}

	maxMembers := req.MaxMembers
	if maxMembers <= 0 {
		maxMembers = 5
	}

	board := models.Board{
		UserID:           userID,
		Title:            req.Title,
		Year:             year,
		GridSize:         gridSize,
		Category:         req.Category,
		BoardType:        boardType,
		MaxMembers:       maxMembers,
		GraceSquareTitle: req.GraceSquareTitle,
		IsDefault:        count == 0,
	}

	if err := database.DB.Create(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create board",
		})
	}

	// Auto-create board member with owner role
	member := models.BoardMember{
		BoardID: board.ID,
		UserID:  userID,
		Role:    "owner",
	}
	database.DB.Create(&member)

	database.DB.Preload("Goals", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).Preload("Members.User").First(&board, board.ID)

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
