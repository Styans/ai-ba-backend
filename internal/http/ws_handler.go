package http

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-ba/internal/domain/models"
	"ai-ba/internal/repository"
	"ai-ba/internal/service"

	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
)

// WS message formats
type wsMsg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type startSessionPayload struct {
	Title string `json:"title"`
}

type userMessagePayload struct {
	SessionID uint   `json:"session_id"`
	Text      string `json:"text"`
}

type aiDonePayload struct {
	Text string `json:"text"`
}

// NewWSHandler возвращает функцию-обработчик для websocket соединений.
// Ожидается, что клиент подключается к /ws/agent?token=<jwt> или передаёт Authorization: Bearer <jwt>.
func NewWSHandler(llm *service.LLMService, msgRepo *repository.MessageRepo, sessRepo *repository.SessionRepo, draftRepo *repository.DraftRepo, jwtSecret string) func(conn *websocket.Conn) {

	return func(conn *websocket.Conn) {
		defer conn.Close()

		// 1) пробуем достать то, что положили в Locals
		var tokenStr string

		if v := conn.Locals("authHeader"); v != nil {
			if s, ok := v.(string); ok {
				tokenStr = s // тут будет "Bearer eyJ..."
			}
		}

		// если в заголовке ничего не было — берём из query ?token=
		if tokenStr == "" {
			if v := conn.Locals("queryToken"); v != nil {
				if s, ok := v.(string); ok {
					tokenStr = s // тут может быть либо "Bearer xxx", либо просто "xxx"
				}
			}
		}

		// 3) срезаем "Bearer "
		if strings.HasPrefix(tokenStr, "Bearer ") {
			tokenStr = strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer "))
		}

		if jwtSecret == "" {
			_ = conn.WriteJSON(map[string]string{"type": "error", "message": "server jwt not configured"})
			return
		}

		// Если токена всё ещё нет — уже тогда fallback на auth-message,
		// как у тебя ниже (этот блок можно оставить как есть).
		if tokenStr == "" {
			// твой код с ReadMessage / type="auth"...
			// ...
		}

		// дальше — твоя же проверка JWT, всё ок:
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || token == nil || !token.Valid {
			_ = conn.WriteJSON(map[string]string{"type": "error", "message": "invalid token"})
			return
		}

		// Extract user id (sub) if available
		var userID uint = 0
		if sub, ok := claims["sub"]; ok {
			// jwt.MapClaims numeric values parsed as float64
			switch v := sub.(type) {
			case float64:
				userID = uint(v)
			case int64:
				userID = uint(v)
			case int:
				userID = uint(v)
			case string:
				// ignore string sub here
			}
		}

		// channels
		send := make(chan interface{}, 8)
		defer close(send)

		// writer goroutine
		go func() {
			for m := range send {
				_ = conn.WriteJSON(m)
			}
		}()

		// main read loop
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				// connection closed or error
				return
			}

			var msg wsMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "invalid json"}}
				continue
			}

			switch msg.Type {
			case "start_session":
				var p startSessionPayload
				_ = json.Unmarshal(msg.Payload, &p)

				// create session in DB
				if sessRepo != nil {
					s := &models.Session{
						UserID:    userID,
						Title:     p.Title,
						Status:    "reviewing",
						CreatedAt: time.Now().Unix(),
					}
					if err := sessRepo.Create(s); err == nil {
						send <- map[string]interface{}{"type": "session_started", "payload": map[string]interface{}{"session_id": s.ID}}
					} else {
						send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "failed to create session"}}
					}
				} else {
					// fallback: return synthetic id 0
					send <- map[string]interface{}{"type": "session_started", "payload": map[string]interface{}{"session_id": 0}}
				}

			case "user_message":
				var p userMessagePayload
				_ = json.Unmarshal(msg.Payload, &p)

				// Save user message to DB (if repo available)
				if msgRepo != nil {
					um := &models.Message{
						SessionID: p.SessionID,
						Author:    "user",
						Text:      p.Text,
						CreatedAt: time.Now().Unix(), // Unix int64
					}
					_ = msgRepo.Save(um)
				}

				// Prepare prompt with system instruction
				systemPrompt := "SYSTEM: You are an AI assistant. If the user asks to create a document, report, or draft, you MUST return a JSON object with the following structure: {\"type\": \"create_document\", \"title\": \"Document Title\", \"content\": \"Markdown Content\"}. Do not wrap it in markdown code blocks. If it's a normal chat, just reply with text.\n\nUser: " + p.Text

				// Call LLM synchronously (simple flow)
				reply, err := llm.Generate(systemPrompt)
				if err != nil {
					send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": fmt.Sprintf("llm error: %v", err)}}
					continue
				}

				// Try to parse reply as JSON document creation
				var docPayload struct {
					Type    string `json:"type"`
					Title   string `json:"title"`
					Content string `json:"content"`
				}

				// Clean up potential markdown code blocks if LLM ignores instruction
				cleanReply := strings.TrimSpace(reply)
				cleanReply = strings.TrimPrefix(cleanReply, "```json")
				cleanReply = strings.TrimPrefix(cleanReply, "```")
				cleanReply = strings.TrimSuffix(cleanReply, "```")

				if err := json.Unmarshal([]byte(cleanReply), &docPayload); err == nil && docPayload.Type == "create_document" {
					// It's a document creation request
					if draftRepo != nil {
						draft := &models.Draft{
							UserID:    userID,
							Title:     docPayload.Title,
							Content:   docPayload.Content,
							CreatedAt: time.Now().Unix(),
							UpdatedAt: time.Now().Unix(),
						}
						if err := draftRepo.Create(draft); err == nil {
							// Notify frontend
							send <- map[string]interface{}{
								"type": "document_created",
								"payload": map[string]interface{}{
									"draft_id": draft.ID,
									"title":    draft.Title,
								},
							}
							// Also save a message saying document was created
							reply = fmt.Sprintf("I've created a document for you: %s", draft.Title)
						} else {
							reply = "Failed to save document."
						}
					}
				}

				// Save AI message to DB (if repo)
				if msgRepo != nil {
					am := &models.Message{
						SessionID: p.SessionID,
						Author:    "ai",
						Text:      reply,
						CreatedAt: time.Now().Unix(), // Unix int64
					}
					_ = msgRepo.Save(am)
				}

				// Send ai_done
				send <- map[string]interface{}{"type": "ai_done", "payload": aiDonePayload{Text: reply}}

			default:
				send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "unknown message type"}}
			}
		}
	}
}
