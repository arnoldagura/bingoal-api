package handlers

import (
	"time"

	"github.com/arnold/bingoals-api/internal/database"
	"github.com/arnold/bingoals-api/internal/middleware"
	"github.com/arnold/bingoals-api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CreateInvite generates an invite code for a board (owner only)
func CreateInvite(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	// Verify user is the board owner
	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found or you are not the owner",
		})
	}

	var req models.CreateInviteRequest
	c.BodyParser(&req) // optional body

	invite := models.BoardInvite{
		BoardID:   boardID,
		InviterID: userID,
		MaxUses:   req.MaxUses,
	}

	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		invite.ExpiresAt = &exp
	}

	if err := database.DB.Create(&invite).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create invite",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(invite)
}

// JoinBoard joins a board via invite code
func JoinBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	code := c.Params("code")

	// Find the invite
	var invite models.BoardInvite
	if err := database.DB.Where("invite_code = ?", code).First(&invite).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Invalid invite code",
		})
	}

	if !invite.IsValid() {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"error": "This invite has expired or reached its usage limit",
		})
	}

	// Check the board exists
	var board models.Board
	if err := database.DB.First(&board, invite.BoardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board no longer exists",
		})
	}

	// Check if already a member
	var existing models.BoardMember
	if err := database.DB.Where("board_id = ? AND user_id = ?", invite.BoardID, userID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "You are already a member of this board",
		})
	}

	// Check member limit
	var memberCount int64
	database.DB.Model(&models.BoardMember{}).Where("board_id = ?", invite.BoardID).Count(&memberCount)
	if int(memberCount) >= board.MaxMembers {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "This board has reached its maximum number of members",
		})
	}

	// Create membership
	member := models.BoardMember{
		BoardID: invite.BoardID,
		UserID:  userID,
		Role:    "member",
	}
	if err := database.DB.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to join board",
		})
	}

	// Increment invite usage
	database.DB.Model(&invite).Update("used_count", invite.UsedCount+1)

	// Log activity
	LogActivity(invite.BoardID, userID, "member_joined", nil, nil)

	// Notify other board members
	var joiner models.User
	database.DB.First(&joiner, userID)
	name := joiner.DisplayName
	if name == "" {
		name = joiner.Name
	}
	notifyBoardMembers(invite.BoardID, userID, "member_joined",
		"New member joined",
		name+" joined "+board.Title,
		map[string]interface{}{"boardId": invite.BoardID.String()},
	)

	// Broadcast member joined via WebSocket
	WS.Broadcast(invite.BoardID, userID, WSEvent{
		Type:    EventMemberJoined,
		BoardID: invite.BoardID.String(),
		UserID:  userID.String(),
		Data: map[string]interface{}{
			"userName": name,
		},
	})

	return c.JSON(fiber.Map{
		"message": "Successfully joined board",
		"boardId": invite.BoardID,
	})
}

// GetMembers lists all members of a board
func GetMembers(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	// Verify user has access to this board
	if !isBoardMember(boardID, userID) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	var members []models.BoardMember
	database.DB.Where("board_id = ?", boardID).
		Preload("User").
		Find(&members)

	// Convert to MemberInfo
	var result []models.MemberInfo
	for _, m := range members {
		result = append(result, models.MemberInfo{
			ID:          m.UserID,
			Name:        m.User.Name,
			DisplayName: m.User.DisplayName,
			AvatarURL:   m.User.AvatarURL,
			Role:        m.Role,
		})
	}

	return c.JSON(result)
}

// RemoveMember removes a member from a board (owner only)
func RemoveMember(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Verify caller is the board owner
	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the board owner can remove members",
		})
	}

	// Can't remove yourself (owner)
	if targetUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Owner cannot be removed. Transfer ownership first or delete the board.",
		})
	}

	result := database.DB.Where("board_id = ? AND user_id = ?", boardID, targetUserID).Delete(&models.BoardMember{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Member not found",
		})
	}

	LogActivity(boardID, targetUserID, "member_left", nil, map[string]interface{}{
		"removedBy": userID,
	})

	// Broadcast member removed via WebSocket
	WS.Broadcast(boardID, userID, WSEvent{
		Type:    EventMemberLeft,
		BoardID: boardID.String(),
		UserID:  targetUserID.String(),
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// LeaveBoard allows a member to leave a board (not the owner)
func LeaveBoard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid board ID",
		})
	}

	// Check if user is the owner — owners can't leave
	var board models.Board
	if err := database.DB.First(&board, boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Board not found",
		})
	}

	if board.UserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Owner cannot leave the board. Transfer ownership first or delete the board.",
		})
	}

	result := database.DB.Where("board_id = ? AND user_id = ?", boardID, userID).Delete(&models.BoardMember{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "You are not a member of this board",
		})
	}

	LogActivity(boardID, userID, "member_left", nil, nil)

	// Broadcast member left via WebSocket
	WS.Broadcast(boardID, userID, WSEvent{
		Type:    EventMemberLeft,
		BoardID: boardID.String(),
		UserID:  userID.String(),
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// isBoardMember checks if a user is a member of a board (owner or member)
func isBoardMember(boardID, userID uuid.UUID) bool {
	// Check ownership first
	var board models.Board
	if err := database.DB.Where("id = ? AND user_id = ?", boardID, userID).First(&board).Error; err == nil {
		return true
	}
	// Check membership
	var member models.BoardMember
	return database.DB.Where("board_id = ? AND user_id = ?", boardID, userID).First(&member).Error == nil
}
