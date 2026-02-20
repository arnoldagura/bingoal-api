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
	if req.Mood != nil {
		goal.Mood = req.Mood
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
		goal.Mood = nil
		goal.Status = "not_started"
		goal.IsCompleted = false
		goal.Progress = 0
		goal.CompletedAt = nil
		// Delete associated mini-goals, reflection, and per-member state
		if !isNew {
			// Delete MiniGoalMember rows for this goal's mini-goals
			var miniGoalIDs []uuid.UUID
			database.DB.Model(&models.MiniGoal{}).Where("goal_id = ?", goal.ID).Pluck("id", &miniGoalIDs)
			if len(miniGoalIDs) > 0 {
				database.DB.Where("mini_goal_id IN ?", miniGoalIDs).Delete(&models.MiniGoalMember{})
			}
			database.DB.Where("goal_id = ?", goal.ID).Delete(&models.MiniGoal{})
			database.DB.Where("goal_id = ?", goal.ID).Delete(&models.Reflection{})
			database.DB.Where("goal_id = ?", goal.ID).Delete(&models.GoalMember{})
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

	if board.BoardType == "shared" {
		return toggleGoalForMember(c, board, goal, userID, position)
	}
	return toggleGoalPersonal(c, board, goal, userID, position)
}

// toggleGoalPersonal handles goal toggling for personal boards (unchanged behavior).
func toggleGoalPersonal(c *fiber.Ctx, board models.Board, goal models.Goal, userID uuid.UUID, position int) error {
	boardID := board.ID

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

	gemsAwarded := 0
	milestones := []string{}
	if goal.Status == "completed" && !wasCompleted {
		gemsAwarded, milestones = calculateMilestonesAndGems(boardID, board.GridSize, position)
		awardGemsAndStreak(userID, gemsAwarded)
		createBlankReflection(goal.ID)
	}

	return c.JSON(fiber.Map{
		"goal":        goal,
		"gemsAwarded": gemsAwarded,
		"milestones":  milestones,
	})
}

// toggleGoalForMember handles goal toggling for shared boards using GoalMember pivot table.
func toggleGoalForMember(c *fiber.Ctx, board models.Board, goal models.Goal, userID uuid.UUID, position int) error {
	boardID := board.ID

	// Find or create GoalMember for this user
	var gm models.GoalMember
	err := database.DB.Where("goal_id = ? AND user_id = ?", goal.ID, userID).First(&gm).Error
	if err != nil {
		gm = models.GoalMember{
			GoalID: goal.ID,
			UserID: userID,
			Status: "not_started",
		}
	}

	var miniGoals []models.MiniGoal
	database.DB.Where("goal_id = ?", goal.ID).Find(&miniGoals)

	wasCompleted := gm.Status == "completed"
	hasMiniGoals := len(miniGoals) > 0

	var nextStatus string
	var completedAt *time.Time
	var isCompleted bool
	now := time.Now()

	if !hasMiniGoals {
		if gm.Status == "completed" {
			nextStatus = "not_started"
			isCompleted = false
			completedAt = nil
		} else {
			nextStatus = "completed"
			isCompleted = true
			completedAt = &now
		}
	} else {
		switch gm.Status {
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

	gm.Status = nextStatus
	gm.IsCompleted = isCompleted
	gm.CompletedAt = completedAt

	if gm.ID == (uuid.UUID{}) {
		if err := database.DB.Create(&gm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to toggle goal",
			})
		}
	} else {
		if err := database.DB.Save(&gm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to toggle goal",
			})
		}
	}

	// Broadcast toggle to other members
	WS.Broadcast(boardID, userID, WSEvent{
		Type:    EventGoalUpdated,
		BoardID: boardID.String(),
		UserID:  userID.String(),
		Data:    goal,
	})

	gemsAwarded := 0
	milestones := []string{}
	if gm.Status == "completed" && !wasCompleted {
		// Calculate milestones from this user's GoalMember rows
		gemsAwarded, milestones = calculateMemberMilestones(boardID, board.GridSize, position, userID)
		awardGemsAndStreak(userID, gemsAwarded)
		createBlankReflection(goal.ID)

		// Log activity + notify other members
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

	// Return goal with this user's status overlaid
	goal.Status = gm.Status
	goal.IsCompleted = gm.IsCompleted
	goal.Progress = gm.Progress
	goal.CompletedAt = gm.CompletedAt

	return c.JSON(fiber.Map{
		"goal":        goal,
		"gemsAwarded": gemsAwarded,
		"milestones":  milestones,
	})
}

// calculateMilestonesAndGems calculates milestones for personal boards using Goal.Status.
func calculateMilestonesAndGems(boardID uuid.UUID, gridSize, position int) (int, []string) {
	var boardGoals []models.Goal
	database.DB.Where("board_id = ?", boardID).Find(&boardGoals)

	completed := make(map[int]bool)
	for _, g := range boardGoals {
		if g.Status == "completed" {
			completed[g.Position] = true
		}
	}

	return checkMilestones(completed, gridSize, position)
}

// calculateMemberMilestones calculates milestones for shared boards using GoalMember rows.
func calculateMemberMilestones(boardID uuid.UUID, gridSize, position int, userID uuid.UUID) (int, []string) {
	// Get all goals for this board
	var boardGoals []models.Goal
	database.DB.Where("board_id = ?", boardID).Find(&boardGoals)

	goalIDs := make([]uuid.UUID, len(boardGoals))
	goalPositions := make(map[uuid.UUID]int)
	for i, g := range boardGoals {
		goalIDs[i] = g.ID
		goalPositions[g.ID] = g.Position
	}

	// Get this user's GoalMember rows
	var goalMembers []models.GoalMember
	database.DB.Where("goal_id IN ? AND user_id = ? AND is_completed = true", goalIDs, userID).Find(&goalMembers)

	completed := make(map[int]bool)
	for _, gm := range goalMembers {
		if pos, ok := goalPositions[gm.GoalID]; ok {
			completed[pos] = true
		}
	}

	return checkMilestones(completed, gridSize, position)
}

// checkMilestones checks row/col/diagonal/corner/blackout completion.
func checkMilestones(completed map[int]bool, gridSize, position int) (int, []string) {
	gemsAwarded := 0
	switch gridSize {
	case 3:
		gemsAwarded = 5
	case 7:
		gemsAwarded = 2
	default:
		gemsAwarded = 3
	}

	milestones := []string{}
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

	return gemsAwarded, milestones
}

// awardGemsAndStreak gives gems and updates streak for a user.
func awardGemsAndStreak(userID uuid.UUID, gemsAwarded int) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return
	}
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

// overlayMemberStatus replaces goal/mini-goal status fields with per-member
// values for shared boards. For personal boards this is a no-op.
func overlayMemberStatus(goals []models.Goal, boardType string, userID uuid.UUID) {
	if boardType != "shared" || len(goals) == 0 {
		return
	}

	// Collect goal IDs
	goalIDs := make([]uuid.UUID, len(goals))
	for i := range goals {
		goalIDs[i] = goals[i].ID
	}

	// Batch-load this user's GoalMember rows
	var goalMembers []models.GoalMember
	database.DB.Where("goal_id IN ? AND user_id = ?", goalIDs, userID).Find(&goalMembers)

	gmMap := make(map[uuid.UUID]*models.GoalMember)
	for i := range goalMembers {
		gmMap[goalMembers[i].GoalID] = &goalMembers[i]
	}

	// Batch-load completed counts per goal (how many members completed each)
	type countResult struct {
		GoalID uuid.UUID
		Count  int
	}
	var counts []countResult
	database.DB.Model(&models.GoalMember{}).
		Select("goal_id, COUNT(*) as count").
		Where("goal_id IN ? AND is_completed = true", goalIDs).
		Group("goal_id").
		Find(&counts)

	countMap := make(map[uuid.UUID]int)
	for _, cr := range counts {
		countMap[cr.GoalID] = cr.Count
	}

	// Collect all mini-goal IDs for batch loading
	var allMiniGoalIDs []uuid.UUID
	for i := range goals {
		for j := range goals[i].MiniGoals {
			allMiniGoalIDs = append(allMiniGoalIDs, goals[i].MiniGoals[j].ID)
		}
	}

	mgmMap := make(map[uuid.UUID]bool)
	if len(allMiniGoalIDs) > 0 {
		var miniGoalMembers []models.MiniGoalMember
		database.DB.Where("mini_goal_id IN ? AND user_id = ?", allMiniGoalIDs, userID).Find(&miniGoalMembers)
		for _, mgm := range miniGoalMembers {
			mgmMap[mgm.MiniGoalID] = mgm.IsComplete
		}
	}

	// Overlay per-member status onto each goal
	for i := range goals {
		gm, exists := gmMap[goals[i].ID]
		if exists {
			goals[i].Status = gm.Status
			goals[i].IsCompleted = gm.IsCompleted
			goals[i].Progress = gm.Progress
			goals[i].CompletedAt = gm.CompletedAt
		} else {
			// No member row yet — show as not started for this user
			goals[i].Status = "not_started"
			goals[i].IsCompleted = false
			goals[i].Progress = 0
			goals[i].CompletedAt = nil
		}

		goals[i].CompletedByCount = countMap[goals[i].ID]

		// Overlay mini-goal completion
		for j := range goals[i].MiniGoals {
			if complete, ok := mgmMap[goals[i].MiniGoals[j].ID]; ok {
				goals[i].MiniGoals[j].IsComplete = complete
			} else {
				goals[i].MiniGoals[j].IsComplete = false
			}
		}
	}
}

// GalleryItem is returned by the GET /api/gallery endpoint
type GalleryItem struct {
	MilestoneID string  `json:"milestoneId"`
	Title       string  `json:"title"`
	ImageURL    *string `json:"imageUrl"`
	IsComplete  bool    `json:"isComplete"`
	GoalTitle   string  `json:"goalTitle"`
	BoardTitle  string  `json:"boardTitle"`
	BoardID     string  `json:"boardId"`
	Position    int     `json:"position"`
	CreatedAt   string  `json:"createdAt"`
}

// GetGallery returns all milestones across all of the user's boards.
func GetGallery(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Fetch all boards the user owns or is a member of
	var boards []models.Board
	database.DB.Where("user_id = ?", userID).Find(&boards)

	// Also include shared boards the user is a member of
	var memberBoards []models.Board
	database.DB.
		Joins("JOIN board_members ON board_members.board_id = boards.id").
		Where("board_members.user_id = ? AND boards.user_id != ?", userID, userID).
		Find(&memberBoards)
	boards = append(boards, memberBoards...)

	if len(boards) == 0 {
		return c.JSON([]GalleryItem{})
	}

	boardIDs := make([]uuid.UUID, len(boards))
	boardMap := make(map[uuid.UUID]models.Board)
	for i, b := range boards {
		boardIDs[i] = b.ID
		boardMap[b.ID] = b
	}

	// Fetch all goals for those boards
	var goals []models.Goal
	database.DB.Where("board_id IN ?", boardIDs).Find(&goals)

	if len(goals) == 0 {
		return c.JSON([]GalleryItem{})
	}

	goalIDs := make([]uuid.UUID, len(goals))
	goalMap := make(map[uuid.UUID]models.Goal)
	for i, g := range goals {
		goalIDs[i] = g.ID
		goalMap[g.ID] = g
	}

	// Fetch all mini-goals for those goals, ordered newest first
	var miniGoals []models.MiniGoal
	database.DB.Where("goal_id IN ?", goalIDs).Order("created_at DESC").Find(&miniGoals)

	items := make([]GalleryItem, 0, len(miniGoals))
	for _, mg := range miniGoals {
		goal := goalMap[mg.GoalID]
		board := boardMap[goal.BoardID]
		goalTitle := ""
		if goal.Title != nil {
			goalTitle = *goal.Title
		}
		items = append(items, GalleryItem{
			MilestoneID: mg.ID.String(),
			Title:       mg.Title,
			ImageURL:    mg.ImageURL,
			IsComplete:  mg.IsComplete,
			GoalTitle:   goalTitle,
			BoardTitle:  board.Title,
			BoardID:     board.ID.String(),
			Position:    goal.Position,
			CreatedAt:   mg.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(items)
}
