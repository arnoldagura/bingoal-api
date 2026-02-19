package handlers

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/arnold/bingoals-api/internal/middleware"
)

// Event types sent over WebSocket
const (
	EventMemberJoined   = "member_joined"
	EventMemberLeft     = "member_left"
	EventGoalUpdated    = "goal_updated"
	EventGoalCompleted  = "goal_completed"
	EventBoardUpdated   = "board_updated"
	EventCommentAdded   = "comment_added"
	EventCommentDeleted = "comment_deleted"
)

// WSEvent is the JSON message sent to connected clients
type WSEvent struct {
	Type    string      `json:"type"`
	BoardID string      `json:"boardId"`
	UserID  string      `json:"userId"`
	Data    interface{} `json:"data,omitempty"`
}

// connection wraps a websocket connection with its user ID
type connection struct {
	conn   *websocket.Conn
	userID uuid.UUID
}

// Hub manages WebSocket connections per board
type Hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[*connection]bool // boardID -> set of connections
}

// Global hub instance
var WS = &Hub{
	rooms: make(map[uuid.UUID]map[*connection]bool),
}

// register adds a connection to a board room
func (h *Hub) register(boardID uuid.UUID, conn *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[boardID] == nil {
		h.rooms[boardID] = make(map[*connection]bool)
	}
	h.rooms[boardID][conn] = true
	log.Printf("WS register: user %s joined board %s (total: %d)", conn.userID, boardID, len(h.rooms[boardID]))
}

// unregister removes a connection from a board room
func (h *Hub) unregister(boardID uuid.UUID, conn *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[boardID]; ok {
		delete(conns, conn)
		log.Printf("WS unregister: user %s left board %s (remaining: %d)", conn.userID, boardID, len(conns))
		if len(conns) == 0 {
			delete(h.rooms, boardID)
		}
	}
}

// Broadcast sends an event to all connections in a board room, excluding the sender
func (h *Hub) Broadcast(boardID uuid.UUID, excludeUserID uuid.UUID, event WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.rooms[boardID]
	if !ok {
		log.Printf("WS broadcast: no connections for board %s", boardID)
		return
	}
	log.Printf("WS broadcast: %s to %d connection(s) on board %s", event.Type, len(conns), boardID)

	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("WS broadcast marshal error: %v", err)
		return
	}

	for c := range conns {
		// Don't send to the user who triggered the event
		if c.userID == excludeUserID {
			continue
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("WS write error: %v", err)
		}
	}
}

// WebSocketUpgrade is the middleware that checks the upgrade request and validates JWT
func WebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		// Authenticate via query param: ?token=<jwt>
		tokenString := c.Query("token")
		if tokenString == "" {
			// Also check Authorization header for non-browser clients
			authHeader := c.Get("Authorization")
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				tokenString = ""
			}
		}

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication token",
			})
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "your-secret-key-change-in-production"
		}

		token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*middleware.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		c.Locals("userId", claims.UserID)
		return c.Next()
	}
}

// HandleWebSocket handles a WebSocket connection for a specific board
func HandleWebSocket(c *websocket.Conn) {
	boardID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		c.Close()
		return
	}

	userID, ok := c.Locals("userId").(uuid.UUID)
	if !ok {
		c.Close()
		return
	}

	conn := &connection{conn: c, userID: userID}
	WS.register(boardID, conn)
	defer WS.unregister(boardID, conn)

	// Keep connection alive — read messages (client sends pings/keepalives)
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}
