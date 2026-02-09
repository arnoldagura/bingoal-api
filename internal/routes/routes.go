package routes

import (
	"github.com/arnold/bingoals-api/internal/handlers"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	api := app.Group("/api")

	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Post("/google", handlers.GoogleLogin)

	protected := api.Group("/", middleware.Protected())

	protected.Get("/me", handlers.GetMe)

	boards := protected.Group("/boards")
	boards.Get("/", handlers.GetBoards)
	boards.Post("/", handlers.CreateBoard)
	boards.Get("/:id", handlers.GetBoard)
	boards.Put("/:id", handlers.UpdateBoard)
	boards.Delete("/:id", handlers.DeleteBoard)

	boards.Put("/:boardId/goals/:position", handlers.UpdateGoal)
	boards.Post("/:boardId/goals/:position/toggle", handlers.ToggleGoalCompletion)

	boards.Post("/:boardId/goals/:position/mini-goals", handlers.CreateMiniGoal)
	boards.Post("/:boardId/goals/:position/mini-goals/:miniGoalId/toggle", handlers.ToggleMiniGoal)
	boards.Put("/:boardId/goals/:position/mini-goals/:miniGoalId", handlers.UpdateMiniGoal)
	boards.Delete("/:boardId/goals/:position/mini-goals/:miniGoalId", handlers.DeleteMiniGoal)

	boards.Get("/:boardId/goals/:position/reflection", handlers.GetReflection)
	boards.Put("/:boardId/goals/:position/reflection", handlers.UpsertReflection)
}
