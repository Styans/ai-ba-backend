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
func NewWSHandler(llm *service.LLMService, draftService *service.DraftService, msgRepo *repository.MessageRepo, sessRepo *repository.SessionRepo) func(conn *websocket.Conn) {
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

				// Fetch history for context
				var history []models.Message
				if msgRepo != nil {
					msgs, _ := msgRepo.GetBySessionID(p.SessionID)
					// Exclude the last message (current one) if it's already saved
					if len(msgs) > 0 && msgs[len(msgs)-1].Text == p.Text {
						history = msgs[:len(msgs)-1]
					} else {
						history = msgs
					}
				}

				// Call LLM with Chat (history)
				fmt.Printf("DEBUG: Chatting with history len=%d, input='%s'\n", len(history), p.Text)

				reply, err := llm.Chat(history, p.Text)
				fmt.Printf("DEBUG: LLM Reply: '%s', Err: %v\n", reply, err)

				if err != nil {
					fmt.Printf("LLM Error: %v\n", err) // Log error to console
					send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "llm error"}}
					continue
				}

				// Try to parse reply as JSON
				cleanReply := strings.TrimSpace(reply)
				start := strings.Index(cleanReply, "{")
				end := strings.LastIndex(cleanReply, "}")
				if start != -1 && end != -1 && end > start {
					cleanReply = cleanReply[start : end+1]
				}

				var jsonReply struct {
					Type string `json:"type"`
				}

				if err := json.Unmarshal([]byte(cleanReply), &jsonReply); err == nil && (jsonReply.Type == "questionnaire" || jsonReply.Type == "requirements") {
					// Handle JSON response
					var rawPayload map[string]interface{}
					_ = json.Unmarshal([]byte(cleanReply), &rawPayload)
					send <- rawPayload

					// Save AI message to DB (as JSON)
					if msgRepo != nil {
						am := &models.Message{
							SessionID: p.SessionID,
							Author:    "ai",
							Text:      cleanReply,
							CreatedAt: time.Now().Unix(),
						}
						_ = msgRepo.Save(am)
					}

					// If requirements, generate document
					if jsonReply.Type == "requirements" {
						// Try to parse as AnalysisData (Final Report)
						// The AI returns { "type": "requirements", "data": { ... } }
						// So we need to unmarshal into a wrapper first
						var wrapper struct {
							Type string               `json:"type"`
							Data service.AnalysisData `json:"data"`
						}

						if err := json.Unmarshal([]byte(cleanReply), &wrapper); err == nil && wrapper.Type == "requirements" {
							detailedReq := wrapper.Data

							// Create or Update Draft
							draft, err := draftService.CreateFromAnalysisData(p.SessionID, userID, &detailedReq)
							if err != nil {
								fmt.Printf("Failed to create draft from analysis data: %v", err)
								send <- map[string]interface{}{
									"type": "error",
									"payload": map[string]interface{}{
										"msg": "Failed to generate document",
									},
								}
							} else {
								send <- map[string]interface{}{
									"type": "doc_generated",
									"payload": map[string]interface{}{
										"draft_id":  draft.ID,
										"file_path": draft.FilePath,
										"url":       fmt.Sprintf("/drafts/%d/download", draft.ID),
										"title":     detailedReq.Project.Name,
									},
								}
							}
						} else {
							// Fallback to old SmartAnalysisData
							// ... (existing fallback logic)
							fmt.Printf("Failed to parse as detailed requirements, trying smart analysis...\n")
							// We can leave the fallback logic as is or just log error if we are sure we want detailed only
							// For now, let's just log error and send error message if detailed parsing failed but type was requirements
							// Actually, the original code had a fallback. Let's keep it simple for now.
							send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "Failed to parse document data"}}

							// Fallback to legacy SmartAnalysisData (if any old prompts still exist or for backward compatibility)
							var smartData service.SmartAnalysisData
							if err := json.Unmarshal([]byte(cleanReply), &smartData); err == nil {
								draft, err := draftService.CreateFromSmartData(p.SessionID, userID, &smartData)
								if err == nil {
									send <- map[string]interface{}{
										"type": "doc_generated",
										"payload": map[string]interface{}{
											"draft_id":  draft.ID,
											"file_path": draft.FilePath,
											"url":       fmt.Sprintf("/drafts/%d/download", draft.ID),
										},
									}
								} else {
									fmt.Printf("Failed to generate doc (smart): %v\n", err)
								}
							}
						}
					}
				} else {
					// Logging for debugging
					if err != nil {
						fmt.Printf("JSON Parse Error: %v\nCleaned Reply: %s\n", err, cleanReply)
					} else if jsonReply.Type != "questionnaire" && jsonReply.Type != "requirements" {
						fmt.Printf("Unknown JSON Type: %s\n", jsonReply.Type)
					}

					// Fallback to text (or legacy [GENERATE_DOC])
					// Check for [GENERATE_DOC] trigger
					if strings.Contains(reply, "[GENERATE_DOC]") {
						// Trigger doc generation (Legacy)
						draft, err := draftService.CreateFromSession(p.SessionID, userID, msgRepo)
						if err != nil {
							reply = "I tried to generate the document, but something went wrong: " + err.Error()
						} else {
							reply = fmt.Sprintf("I have generated the Business Analysis Document based on our conversation. You can download it here: /drafts/%d/download", draft.ID)
							// Send special event
							send <- map[string]interface{}{
								"type": "doc_generated",
								"payload": map[string]interface{}{
									"draft_id":  draft.ID,
									"file_path": draft.FilePath,
									"url":       fmt.Sprintf("/drafts/%d/download", draft.ID),
								},
							}
						}
					}

					// Save AI message to DB
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
				}

			default:
				send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "unknown message type"}}
			}
		}
	}
}
