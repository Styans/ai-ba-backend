package http

import (
	"encoding/json"
	"fmt"
	"os"
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
func NewWSHandler(llm *service.LLMService, msgRepo *repository.MessageRepo, sessRepo *repository.SessionRepo) func(conn *websocket.Conn) {
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))

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

				// Call LLM synchronously (simple flow)
				reply, err := llm.Generate(p.Text)
				if err != nil {
					send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "llm error"}}
					continue
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
