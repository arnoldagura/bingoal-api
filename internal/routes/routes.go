package routes

import (
	"github.com/arnold/bingoals-api/internal/handlers"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	api := app.Group("/api")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Post("/google", handlers.GoogleLogin)

	// Protected routes
	protected := api.Group("/", middleware.Protected())

	// User routes
	protected.Get("/me", handlers.GetMe)

	// Board routes
	boards := protected.Group("/boards")
	boards.Get("/", handlers.GetBoards)
	boards.Post("/", handlers.CreateBoard)
	boards.Get("/:id", handlers.GetBoard)
	boards.Put("/:id", handlers.UpdateBoard)
	boards.Delete("/:id", handlers.DeleteBoard)

	// Goal routes
	boards.Put("/:boardId/goals/:position", handlers.UpdateGoal)
	boards.Post("/:boardId/goals/:position/toggle", handlers.ToggleGoalCompletion)
}
