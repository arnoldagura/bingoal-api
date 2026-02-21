package handlers

import (
	"sort"
	"time"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// JournalEntry represents a single timeline item in the journal.
type JournalEntry struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // goal_completed, milestone_reached, reflection_added
	GoalTitle  string    `json:"goalTitle"`
	BoardTitle string    `json:"boardTitle"`
	BoardID    string    `json:"boardId"`
	Content    string    `json:"content"`
	ImageURL   *string   `json:"imageUrl"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetJournal returns a chronological timeline of the user's goal activity.
func GetJournal(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Collect all board IDs the user owns or is a member of.
	var ownedIDs []uuid.UUID
	database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Pluck("id", &ownedIDs)

	var memberIDs []uuid.UUID
	database.DB.Model(&models.BoardMember{}).Where("user_id = ?", userID).Pluck("board_id", &memberIDs)

	allBoardIDs := append(ownedIDs, memberIDs...)
	if len(allBoardIDs) == 0 {
		return c.JSON([]JournalEntry{})
	}

	// Build board title lookup.
	var boards []models.Board
	database.DB.Where("id IN ?", allBoardIDs).Select("id, title").Find(&boards)
	boardTitle := map[string]string{}
	for _, b := range boards {
		boardTitle[b.ID.String()] = b.Title
	}

	// Fetch all goals for these boards (lightweight — no preloads yet).
	type goalRow struct {
		ID      uuid.UUID
		BoardID uuid.UUID
		Title   *string
	}
	var goalRows []goalRow
	database.DB.Model(&models.Goal{}).
		Where("board_id IN ?", allBoardIDs).
		Select("id, board_id, title").
		Scan(&goalRows)

	goalBoardID := map[string]uuid.UUID{}
	goalTitleMap := map[string]*string{}
	var allGoalIDs []uuid.UUID
	for _, g := range goalRows {
		goalBoardID[g.ID.String()] = g.BoardID
		goalTitleMap[g.ID.String()] = g.Title
		allGoalIDs = append(allGoalIDs, g.ID)
	}

	var entries []JournalEntry

	// ── 1. Completed goals ───────────────────────────────────────────────────
	var completedGoals []models.Goal
	database.DB.
		Preload("Memories").
		Where("board_id IN ? AND is_completed = true", allBoardIDs).
		Order("completed_at DESC").
		Limit(60).
		Find(&completedGoals)

	for _, g := range completedGoals {
		if g.Title == nil {
			continue
		}
		ts := g.UpdatedAt
		if g.CompletedAt != nil {
			ts = *g.CompletedAt
		}
		var imgURL *string
		if g.ImageURL != nil {
			imgURL = g.ImageURL
		}
		for _, m := range g.Memories {
			if m.IsBoardImage {
				url := m.ImageURL
				imgURL = &url
				break
			}
		}
		entries = append(entries, JournalEntry{
			ID:         "goal_" + g.ID.String(),
			Type:       "goal_completed",
			GoalTitle:  *g.Title,
			BoardTitle: boardTitle[g.BoardID.String()],
			BoardID:    g.BoardID.String(),
			ImageURL:   imgURL,
			Timestamp:  ts,
		})
	}

	// ── 2. Completed mini-goals (milestones) ────────────────────────────────
	if len(allGoalIDs) > 0 {
		var miniGoals []models.MiniGoal
		database.DB.
			Where("goal_id IN ? AND is_complete = true", allGoalIDs).
			Order("updated_at DESC").
			Limit(40).
			Find(&miniGoals)

		for _, mg := range miniGoals {
			gid := mg.GoalID.String()
			bid := goalBoardID[gid]
			gt := ""
			if t := goalTitleMap[gid]; t != nil {
				gt = *t
			}
			entries = append(entries, JournalEntry{
				ID:         "mini_" + mg.ID.String(),
				Type:       "milestone_reached",
				GoalTitle:  gt,
				BoardTitle: boardTitle[bid.String()],
				BoardID:    bid.String(),
				Content:    mg.Title,
				Timestamp:  mg.UpdatedAt,
			})
		}
	}

	// ── 3. Reflections ───────────────────────────────────────────────────────
	if len(allGoalIDs) > 0 {
		var reflections []models.Reflection
		database.DB.
			Where("goal_id IN ? AND (reflection_answer IS NOT NULL OR victories IS NOT NULL OR notes IS NOT NULL)", allGoalIDs).
			Order("updated_at DESC").
			Limit(30).
			Find(&reflections)

		for _, r := range reflections {
			content := ""
			if r.ReflectionAnswer != nil {
				content = *r.ReflectionAnswer
			} else if r.Victories != nil {
				content = *r.Victories
			} else if r.Notes != nil {
				content = *r.Notes
			}
			if content == "" {
				continue
			}
			gid := r.GoalID.String()
			bid := goalBoardID[gid]
			gt := ""
			if t := goalTitleMap[gid]; t != nil {
				gt = *t
			}
			entries = append(entries, JournalEntry{
				ID:         "reflection_" + r.ID.String(),
				Type:       "reflection_added",
				GoalTitle:  gt,
				BoardTitle: boardTitle[bid.String()],
				BoardID:    bid.String(),
				Content:    content,
				Timestamp:  r.UpdatedAt,
			})
		}
	}

	// Sort newest first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if entries == nil {
		entries = []JournalEntry{}
	}
	return c.JSON(entries)
}
