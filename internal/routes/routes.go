package routes

import (
	"github.com/arnold/bingoals-api/internal/handlers"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func Setup(app *fiber.App) {
	api := app.Group("/api")

	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Post("/google", handlers.GoogleLogin)

	protected := api.Group("/", middleware.Protected())

	protected.Get("/me", handlers.GetMe)
	protected.Put("/me", handlers.UpdateProfile)
	protected.Get("/users/:id", handlers.GetUserProfile)

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

	// Board invites & members
	boards.Post("/:id/invites", handlers.CreateInvite)
	boards.Get("/:id/members", handlers.GetMembers)
	boards.Delete("/:id/members/:userId", handlers.RemoveMember)
	boards.Post("/:id/leave", handlers.LeaveBoard)

	// Board activity
	boards.Get("/:id/activity", handlers.GetBoardActivity)

	// Join board via invite code
	protected.Post("/invites/:code/join", handlers.JoinBoard)

	// Goal reactions
	goals := protected.Group("/goals")
	goals.Post("/:id/reactions", handlers.AddReaction)
	goals.Get("/:id/reactions", handlers.GetGoalReactions)
	goals.Post("/:id/comments", handlers.AddComment)
	goals.Get("/:id/comments", handlers.GetGoalComments)
	goals.Delete("/:id/comments/:commentId", handlers.DeleteComment)

	// Notifications
	notifications := protected.Group("/notifications")
	notifications.Get("/", handlers.GetNotifications)
	notifications.Put("/:id/read", handlers.MarkNotificationRead)
	notifications.Post("/read-all", handlers.MarkAllRead)

	// Device token for push notifications
	protected.Post("/device-token", handlers.RegisterDeviceToken)

	// File upload
	protected.Post("/upload", handlers.UploadImage)

	// WebSocket for real-time board updates
	app.Use("/ws", handlers.WebSocketUpgrade())
	app.Get("/ws/boards/:id", websocket.New(handlers.HandleWebSocket))
}
