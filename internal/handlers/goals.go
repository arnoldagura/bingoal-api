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
	if err != nil || position < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position",
		})
	}
	var board models.Board
	if err := database.DB.First(&board, boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Check access: owner or member
	if board.UserID != userID && !isBoardMember(boardID, userID) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	maxPosition := board.GridSize*board.GridSize - 1
	if position > maxPosition {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position for this board's grid size",
		})
	}

	var goal models.Goal
	isNew := false
	if err := database.DB.Where("board_id = ? AND position = ?", boardID, position).First(&goal).Error; err != nil {
		
		goal = models.Goal{
			BoardID:  boardID,
			Position: position,
		}
		isNew = true
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
	if req.Icon != nil {
		goal.Icon = req.Icon
	}
	if req.ImageURL != nil {
		goal.ImageURL = req.ImageURL
	}
	if req.IsCompleted != nil {
		goal.IsCompleted = *req.IsCompleted
		if *req.IsCompleted {
			goal.Status = "completed"
			now := time.Now()
			goal.CompletedAt = &now
		} else {
			goal.Status = "not_started"
			goal.CompletedAt = nil
		}
	}

	// Clear goal: when title is set to empty string, reset everything
	if req.Title != nil && *req.Title == "" {
		goal.Description = nil
		goal.Icon = nil
		goal.ImageURL = nil
		goal.Status = "not_started"
		goal.IsCompleted = false
		goal.Progress = 0
		goal.CompletedAt = nil
		// Delete associated mini-goals and reflection
		if !isNew {
			database.DB.Where("goal_id = ?", goal.ID).Delete(&models.MiniGoal{})
			database.DB.Where("goal_id = ?", goal.ID).Delete(&models.Reflection{})
		}
	}

	if isNew {
		if err := database.DB.Create(&goal).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create goal",
			})
		}
	} else {
		if err := database.DB.Save(&goal).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update goal",
			})
		}
	}

	// Broadcast goal update to other connected clients
	if board.BoardType == "shared" {
		WS.Broadcast(boardID, userID, WSEvent{
			Type:    EventGoalUpdated,
			BoardID: boardID.String(),
			UserID:  userID.String(),
			Data:    goal,
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
	if err != nil || position < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position",
		})
	}

	var board models.Board
	if err := database.DB.First(&board, boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Check access: owner or member
	if board.UserID != userID && !isBoardMember(boardID, userID) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	maxPosition := board.GridSize*board.GridSize - 1
	if position > maxPosition {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position for this board's grid size",
		})
	}

	var goal models.Goal
	if err := database.DB.Where("board_id = ? AND position = ?", boardID, position).First(&goal).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Goal not found",
		})
	}

	var miniGoals []models.MiniGoal
	database.DB.Where("goal_id = ?", goal.ID).Find(&miniGoals)

	wasCompleted := goal.Status == "completed"

	hasMiniGoals := len(miniGoals) > 0

	var nextStatus string
	var completedAt *time.Time
	var isCompleted bool

	now := time.Now()

	if !hasMiniGoals {
		if goal.Status == "completed" {
			nextStatus = "not_started"
			isCompleted = false
			completedAt = nil
		} else {
			nextStatus = "completed"
			isCompleted = true
			completedAt = &now
		}
	} else {
		switch goal.Status {
		case "in_progress":
			nextStatus = "completed"
			isCompleted = true
			completedAt = &now
		case "completed":
			nextStatus = "not_started"
			isCompleted = false
			completedAt = nil
		default:
			nextStatus = "in_progress"
			isCompleted = false
			completedAt = nil
		}
	}

	goal.Status = nextStatus
	goal.IsCompleted = isCompleted
	goal.CompletedAt = completedAt

	if err := database.DB.Save(&goal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to toggle goal",
		})
	}

	// Broadcast toggle for shared boards (all state changes)
	if board.BoardType == "shared" {
		WS.Broadcast(boardID, userID, WSEvent{
			Type:    EventGoalUpdated,
			BoardID: boardID.String(),
			UserID:  userID.String(),
			Data:    goal,
		})
	}

	gemsAwarded := 0
	milestones := []string{}
	if goal.Status == "completed" && !wasCompleted {
		
		switch board.GridSize {
		case 3:
			gemsAwarded = 5
		case 7:
			gemsAwarded = 2
		default:
			gemsAwarded = 3
		}

		
		var boardGoals []models.Goal
		database.DB.Where("board_id = ?", boardID).Find(&boardGoals)

		gridSize := board.GridSize

		
		completed := make(map[int]bool)
		for _, g := range boardGoals {
			if g.Status == "completed" {
				completed[g.Position] = true
			}
		}

		milestones = []string{}
		row := position / gridSize
		col := position % gridSize

		
		rowDone := true
		for c := 0; c < gridSize; c++ {
			if !completed[row*gridSize+c] {
				rowDone = false
				break
			}
		}
		if rowDone {
			milestones = append(milestones, "row")
			gemsAwarded += 10
		}

		
		colDone := true
		for r := 0; r < gridSize; r++ {
			if !completed[r*gridSize+col] {
				colDone = false
				break
			}
		}
		if colDone {
			milestones = append(milestones, "column")
			gemsAwarded += 10
		}

		
		if row == col {
			diagDone := true
			for i := 0; i < gridSize; i++ {
				if !completed[i*gridSize+i] {
					diagDone = false
					break
				}
			}
			if diagDone {
				milestones = append(milestones, "diagonal")
				gemsAwarded += 10
			}
		}

		
		if row+col == gridSize-1 {
			antiDone := true
			for i := 0; i < gridSize; i++ {
				if !completed[i*gridSize+(gridSize-1-i)] {
					antiDone = false
					break
				}
			}
			if antiDone {
				milestones = append(milestones, "anti-diagonal")
				gemsAwarded += 10
			}
		}

		
		corners := []int{0, gridSize - 1, (gridSize - 1) * gridSize, gridSize*gridSize - 1}
		cornersDone := true
		for _, p := range corners {
			if !completed[p] {
				cornersDone = false
				break
			}
		}
		if cornersDone {
			milestones = append(milestones, "corners")
			gemsAwarded += 15
		}

		
		if len(completed) == gridSize*gridSize {
			milestones = append(milestones, "blackout")
			gemsAwarded += 50
		}

		
		var user models.User
		if err := database.DB.First(&user, userID).Error; err == nil {
			user.TotalGems += gemsAwarded

			today := time.Now().Truncate(24 * time.Hour)
			if user.LastActiveDate != nil {
				lastActive := user.LastActiveDate.Truncate(24 * time.Hour)
				daysSince := int(today.Sub(lastActive).Hours() / 24)
				if daysSince == 1 {
					user.DailyStreak++
				} else if daysSince > 1 {
					user.DailyStreak = 1
				}
				
			} else {
				user.DailyStreak = 1
			}
			user.LastActiveDate = &today

			database.DB.Save(&user)
		}

		
		createBlankReflection(goal.ID)

		// Set completedBy for shared boards
		goal.CompletedBy = &userID
		database.DB.Save(&goal)

		// Log activity + notify for shared boards
		if board.BoardType == "shared" {
			goalTitle := ""
			if goal.Title != nil {
				goalTitle = *goal.Title
			}
			LogActivity(boardID, userID, "goal_completed", &goal.ID, map[string]interface{}{
				"goalTitle": goalTitle,
				"position":  goal.Position,
			})

			var completer models.User
			database.DB.First(&completer, userID)
			name := completer.DisplayName
			if name == "" {
				name = completer.Name
			}
			notifyBoardMembers(boardID, userID, "goal_completed",
				"Goal completed!",
				name+" completed \""+goalTitle+"\" on "+board.Title,
				map[string]interface{}{"boardId": boardID.String(), "goalId": goal.ID.String()},
			)

			// Broadcast via WebSocket
			WS.Broadcast(boardID, userID, WSEvent{
				Type:    EventGoalCompleted,
				BoardID: boardID.String(),
				UserID:  userID.String(),
				Data: map[string]interface{}{
					"goalTitle": goalTitle,
					"position":  goal.Position,
					"userName":  name,
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"goal":        goal,
		"gemsAwarded": gemsAwarded,
		"milestones":  milestones,
	})
}
