package handlers

import (
	"math/rand"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetReflection(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	var reflection models.Reflection
	if err := database.DB.Where("goal_id = ?", goal.ID).First(&reflection).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No reflection found for this goal",
		})
	}

	return c.JSON(reflection)
}

func UpsertReflection(c *fiber.Ctx) error {
	goal, _, fiberErr := findGoalByBoardAndPosition(c)
	if fiberErr != nil {
		return fiberErr
	}

	var req models.UpsertReflectionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var reflection models.Reflection
	err := database.DB.Where("goal_id = ?", goal.ID).First(&reflection).Error
	if err != nil {
		
		prompt := models.ReflectionPrompts[rand.Intn(len(models.ReflectionPrompts))]
		reflection = models.Reflection{
			GoalID:           goal.ID,
			ReflectionPrompt: &prompt,
		}
	}

	if req.Obstacles != nil {
		reflection.Obstacles = req.Obstacles
	}
	if req.Victories != nil {
		reflection.Victories = req.Victories
	}
	if req.Notes != nil {
		reflection.Notes = req.Notes
	}
	if req.ReflectionAnswer != nil {
		reflection.ReflectionAnswer = req.ReflectionAnswer
	}

	if err := database.DB.Save(&reflection).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save reflection",
		})
	}

	return c.JSON(reflection)
}


func createBlankReflection(goalID uuid.UUID) {

	var existing models.Reflection
	if database.DB.Where("goal_id = ?", goalID).First(&existing).Error == nil {
		return
	}

	prompt := models.ReflectionPrompts[rand.Intn(len(models.ReflectionPrompts))]
	reflection := models.Reflection{
		GoalID:           goalID,
		ReflectionPrompt: &prompt,
	}
	database.DB.Create(&reflection)
}
