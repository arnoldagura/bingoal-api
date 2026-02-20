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


func recalculateGoalProgress(goalID uuid.UUID) {
	var miniGoals []models.MiniGoal
	database.DB.Where("goal_id = ?", goalID).Find(&miniGoals)

	progress := 0
	for _, mg := range miniGoals {
		if mg.IsComplete {
			progress += mg.Percentage
		}
	}
	if progress > 100 {
		progress = 100
	}

	updates := map[string]interface{}{"progress": progress}
	if progress >= 100 {
		updates["status"] = "completed"
		updates["is_completed"] = true
		now := time.Now()
		updates["completed_at"] = &now
	} else if progress > 0 {
		updates["status"] = "in_progress"
		updates["is_completed"] = false
		updates["completed_at"] = nil
	} else {
		updates["status"] = "not_started"
		updates["is_completed"] = false
		updates["completed_at"] = nil
	}

	database.DB.Model(&models.Goal{}).Where("id = ?", goalID).Updates(updates)
}


func findGoalByBoardAndPosition(c *fiber.Ctx) (*models.Goal, *models.Board, error) {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("boardId"))
	if err != nil {
		return nil, nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	position, err := strconv.Atoi(c.Params("position"))
	if err != nil || position < 0 {
		return nil, nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid position",
		})
	}

	var board models.Board
	if err := database.DB.Where("id = ?", boardID).First(&board).Error; err != nil {
		return nil, nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	// Check access: owner or member
	if board.UserID != userID && !isBoardMember(boardID, userID) {
		return nil, nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	var goal models.Goal
	if err := database.DB.Where("board_id = ? AND position = ?", boardID, position).First(&goal).Error; err != nil {
		return nil, nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Goal not found",
		})
	}

	return &goal, &board, nil
}

func CreateMiniGoal(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	var req models.CreateMiniGoalRequest
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
	if req.Percentage < 1 || req.Percentage > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Percentage must be between 1 and 100",
		})
	}

	// Check total percentage doesn't exceed 100
	var existing []models.MiniGoal
	database.DB.Where("goal_id = ?", goal.ID).Find(&existing)
	totalPct := req.Percentage
	for _, mg := range existing {
		totalPct += mg.Percentage
	}
	if totalPct > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Total percentage would exceed 100",
		})
	}

	miniGoal := models.MiniGoal{
		GoalID:     goal.ID,
		Title:      req.Title,
		Percentage: req.Percentage,
	}

	if err := database.DB.Create(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create mini-goal",
		})
	}

	recalculateGoalProgress(goal.ID)

	return c.Status(fiber.StatusCreated).JSON(miniGoal)
}

func ToggleMiniGoal(c *fiber.Ctx) error {
	goal, board, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	miniGoalID, err := uuid.Parse(c.Params("miniGoalId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid mini-goal ID",
		})
	}

	var miniGoal models.MiniGoal
	if err := database.DB.Where("id = ? AND goal_id = ?", miniGoalID, goal.ID).First(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Mini-goal not found",
		})
	}

	if board.BoardType == "shared" {
		userID := middleware.GetUserID(c)

		// Find or create MiniGoalMember for this user
		var mgm models.MiniGoalMember
		err := database.DB.Where("mini_goal_id = ? AND user_id = ?", miniGoal.ID, userID).First(&mgm).Error
		if err != nil {
			mgm = models.MiniGoalMember{
				MiniGoalID: miniGoal.ID,
				UserID:     userID,
			}
		}

		mgm.IsComplete = !mgm.IsComplete

		if mgm.ID == (uuid.UUID{}) {
			database.DB.Create(&mgm)
		} else {
			database.DB.Save(&mgm)
		}

		recalculateGoalProgressForMember(goal.ID, userID)

		// Return mini-goal with this user's completion overlaid
		miniGoal.IsComplete = mgm.IsComplete
		return c.JSON(miniGoal)
	}

	// Personal board: toggle directly on the mini-goal
	miniGoal.IsComplete = !miniGoal.IsComplete
	if err := database.DB.Save(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to toggle mini-goal",
		})
	}

	recalculateGoalProgress(goal.ID)

	return c.JSON(miniGoal)
}

// recalculateGoalProgressForMember recalculates per-member goal progress from MiniGoalMember rows.
func recalculateGoalProgressForMember(goalID, userID uuid.UUID) {
	var miniGoals []models.MiniGoal
	database.DB.Where("goal_id = ?", goalID).Find(&miniGoals)

	// Get this user's MiniGoalMember rows
	miniGoalIDs := make([]uuid.UUID, len(miniGoals))
	for i, mg := range miniGoals {
		miniGoalIDs[i] = mg.ID
	}

	memberComplete := make(map[uuid.UUID]bool)
	if len(miniGoalIDs) > 0 {
		var mgms []models.MiniGoalMember
		database.DB.Where("mini_goal_id IN ? AND user_id = ?", miniGoalIDs, userID).Find(&mgms)
		for _, mgm := range mgms {
			memberComplete[mgm.MiniGoalID] = mgm.IsComplete
		}
	}

	progress := 0
	for _, mg := range miniGoals {
		if memberComplete[mg.ID] {
			progress += mg.Percentage
		}
	}
	if progress > 100 {
		progress = 100
	}

	// Find or create GoalMember
	var gm models.GoalMember
	err := database.DB.Where("goal_id = ? AND user_id = ?", goalID, userID).First(&gm).Error
	if err != nil {
		gm = models.GoalMember{GoalID: goalID, UserID: userID}
	}

	gm.Progress = progress
	if progress >= 100 {
		gm.Status = "completed"
		gm.IsCompleted = true
		now := time.Now()
		gm.CompletedAt = &now
	} else if progress > 0 {
		gm.Status = "in_progress"
		gm.IsCompleted = false
		gm.CompletedAt = nil
	} else {
		gm.Status = "not_started"
		gm.IsCompleted = false
		gm.CompletedAt = nil
	}

	if gm.ID == (uuid.UUID{}) {
		database.DB.Create(&gm)
	} else {
		database.DB.Save(&gm)
	}
}

func UpdateMiniGoal(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	miniGoalID, err := uuid.Parse(c.Params("miniGoalId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid mini-goal ID",
		})
	}

	var miniGoal models.MiniGoal
	if err := database.DB.Where("id = ? AND goal_id = ?", miniGoalID, goal.ID).First(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Mini-goal not found",
		})
	}

	var req models.UpdateMiniGoalRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title != nil {
		miniGoal.Title = *req.Title
	}
	if req.ImageURL != nil {
		miniGoal.ImageURL = req.ImageURL
	}
	if req.Percentage != nil {
		if *req.Percentage < 1 || *req.Percentage > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Percentage must be between 1 and 100",
			})
		}
		// Check total percentage excluding this mini-goal
		var others []models.MiniGoal
		database.DB.Where("goal_id = ? AND id != ?", goal.ID, miniGoalID).Find(&others)
		totalPct := *req.Percentage
		for _, mg := range others {
			totalPct += mg.Percentage
		}
		if totalPct > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Total percentage would exceed 100",
			})
		}
		miniGoal.Percentage = *req.Percentage
	}

	if err := database.DB.Save(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update mini-goal",
		})
	}

	recalculateGoalProgress(goal.ID)

	return c.JSON(miniGoal)
}

func DeleteMiniGoal(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	miniGoalID, err := uuid.Parse(c.Params("miniGoalId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid mini-goal ID",
		})
	}

	var miniGoal models.MiniGoal
	if err := database.DB.Where("id = ? AND goal_id = ?", miniGoalID, goal.ID).First(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Mini-goal not found",
		})
	}

	if err := database.DB.Delete(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete mini-goal",
		})
	}

	recalculateGoalProgress(goal.ID)

	return c.SendStatus(fiber.StatusNoContent)
}
