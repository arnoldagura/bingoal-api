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
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
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

	miniGoal.IsComplete = !miniGoal.IsComplete
	if err := database.DB.Save(&miniGoal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to toggle mini-goal",
		})
	}

	recalculateGoalProgress(goal.ID)

	return c.JSON(miniGoal)
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
