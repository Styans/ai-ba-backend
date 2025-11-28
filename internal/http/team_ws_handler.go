package http

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ai-ba/internal/domain/models"
	"ai-ba/internal/repository"

	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients mapped by UserID
	clients map[uint]*websocket.Conn
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint]*websocket.Conn),
	}
}

func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[userID] = conn
}

func (h *Hub) Unregister(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[userID]; ok {
		delete(h.clients, userID)
	}
}

func (h *Hub) SendTo(userID uint, msg interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conn, ok := h.clients[userID]; ok {
		_ = conn.WriteJSON(msg)
	}
}

type teamWsMsg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type privateMessagePayload struct {
	ReceiverID uint   `json:"receiver_id"`
	Content    string `json:"content"`
}

type historyRequestPayload struct {
	OtherUserID uint `json:"other_user_id"`
}

func NewTeamWSHandler(hub *Hub, msgRepo *repository.TeamMessageRepo, jwtSecret string) func(conn *websocket.Conn) {
	return func(conn *websocket.Conn) {
		// 1. Auth (simplified copy from agent handler)
		tokenStr := conn.Query("token")
		if tokenStr == "" {
			_ = conn.WriteJSON(map[string]string{"type": "error", "message": "missing token"})
			conn.Close()
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			_ = conn.WriteJSON(map[string]string{"type": "error", "message": "invalid token"})
			conn.Close()
			return
		}

		var userID uint
		if sub, ok := claims["sub"].(float64); ok {
			userID = uint(sub)
		} else {
			conn.Close()
			return
		}

		// Register
		hub.Register(userID, conn)
		defer func() {
			hub.Unregister(userID)
			conn.Close()
		}()

		// Read loop
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg teamWsMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "private_message":
				fmt.Printf("Received private_message from %d\n", userID)
				var p privateMessagePayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					fmt.Printf("Parsed payload: %+v\n", p)
					// Save to DB
					tm := &models.TeamMessage{
						SenderID:   userID,
						ReceiverID: p.ReceiverID,
						Content:    p.Content,
						CreatedAt:  time.Now(),
					}
					if err := msgRepo.Save(tm); err != nil {
						fmt.Printf("Error saving message: %v\n", err)
					} else {
						fmt.Printf("Message saved with ID: %d\n", tm.ID)
					}

					// Send to receiver
					fmt.Printf("Sending to receiver %d\n", p.ReceiverID)
					hub.SendTo(p.ReceiverID, map[string]interface{}{
						"type": "new_message",
						"payload": map[string]interface{}{
							"id":          tm.ID,
							"sender_id":   tm.SenderID,
							"receiver_id": tm.ReceiverID,
							"content":     tm.Content,
							"created_at":  tm.CreatedAt,
						},
					})

					// Echo back to sender (so they know it's sent/saved)
					hub.SendTo(userID, map[string]interface{}{
						"type": "message_sent",
						"payload": map[string]interface{}{
							"id":          tm.ID,
							"sender_id":   tm.SenderID,
							"receiver_id": tm.ReceiverID,
							"content":     tm.Content,
							"created_at":  tm.CreatedAt,
						},
					})
				} else {
					fmt.Printf("Error unmarshaling payload: %v\n", err)
				}

			case "get_history":
				var p historyRequestPayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					history, _ := msgRepo.GetHistory(userID, p.OtherUserID)
					conn.WriteJSON(map[string]interface{}{
						"type": "history",
						"payload": map[string]interface{}{
							"other_user_id": p.OtherUserID,
							"messages":      history,
						},
					})
				}
			}
		}
	}
}
