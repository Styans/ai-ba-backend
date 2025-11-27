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
				systemPrompt := "SYSTEM: You are an AI Business Analyst. \n" +
					"If the user asks to create a document, return JSON: {\"type\": \"create_document\", \"title\": \"Title\", \"content\": \"Markdown Content\"}.\n" +
					"If the user asks to edit/update an existing document, return JSON: {\"type\": \"update_document\", \"title\": \"Title\", \"content\": \"Updated Markdown Content\"}.\n" +
					"If you need to gather detailed information from the user (e.g. for requirements gathering), return JSON: {\"type\": \"questionnaire\", \"questions\": [{\"id\": \"1\", \"text\": \"Question?\"}]}.\n" +
					"IMPORTANT: Return ONLY valid JSON. Do not wrap in markdown code blocks. Check for syntax errors (e.g. missing quotes, commas). \n" +
					"For normal chat, just reply with text.\n" +
					"User: " + p.Text

				// Call LLM synchronously (simple flow)
				reply, err := llm.Generate(systemPrompt)
				if err != nil {
					send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": fmt.Sprintf("llm error: %v", err)}}
					continue
				}

				// Try to parse reply as JSON document creation
				var docPayload struct {
					Type      string `json:"type"`
					Title     string `json:"title"`
					Content   string `json:"content"`
					Questions []struct {
						ID   string `json:"id"`
						Text string `json:"text"`
					} `json:"questions"`
				}

				// Clean up potential markdown code blocks if LLM ignores instruction
				cleanReply := strings.TrimSpace(reply)
				// More robust JSON extraction: find first '{' and last '}'
				start := strings.Index(cleanReply, "{")
				end := strings.LastIndex(cleanReply, "}")

				isJSON := false
				if start != -1 && end != -1 && end > start {
					jsonStr := cleanReply[start : end+1]
					if err := json.Unmarshal([]byte(jsonStr), &docPayload); err == nil {
						isJSON = true
					}
				}

				if isJSON {
					if docPayload.Type == "create_document" || docPayload.Type == "update_document" {
						if draftRepo != nil {
							// Check if a draft already exists for this session
							existingDraft, err := draftRepo.GetBySessionID(p.SessionID)

							if err == nil && existingDraft != nil {
								// UPDATE existing draft
								existingDraft.Title = docPayload.Title
								existingDraft.Content = docPayload.Content
								existingDraft.UpdatedAt = time.Now().Unix()

								if err := draftRepo.Update(existingDraft); err == nil {
									send <- map[string]interface{}{
										"type": "document_created", // We keep this type for frontend compatibility for now
										"payload": map[string]interface{}{
											"draft_id":   existingDraft.ID,
											"session_id": existingDraft.SessionID,
											"title":      existingDraft.Title,
										},
									}
									reply = fmt.Sprintf("I've updated the document: %s", existingDraft.Title)
								} else {
									reply = "Failed to update document."
								}
							} else {
								// CREATE new draft
								draft := &models.Draft{
									UserID:    userID,
									SessionID: p.SessionID,
									Title:     docPayload.Title,
									Content:   docPayload.Content,
									CreatedAt: time.Now().Unix(),
									UpdatedAt: time.Now().Unix(),
								}
								if err := draftRepo.Create(draft); err == nil {
									// Update Session Title if it's "New Project" or generic
									// We can just always update it to match the main document of the session
									if sessRepo != nil {
										session, err := sessRepo.GetByID(p.SessionID)
										if err == nil && (session.Title == "New Project" || session.Title == "") {
											session.Title = draft.Title
											_ = sessRepo.Update(session)
										}
									}

									send <- map[string]interface{}{
										"type": "document_created",
										"payload": map[string]interface{}{
											"draft_id":   draft.ID,
											"session_id": draft.SessionID,
											"title":      draft.Title,
										},
									}
									reply = fmt.Sprintf("I've created a document for you: %s", draft.Title)
								} else {
									reply = "Failed to save document."
								}
							}
						}
						// Send ai_done with the reply text
						send <- map[string]interface{}{"type": "ai_done", "payload": aiDonePayload{Text: reply}}

					} else if docPayload.Type == "questionnaire" {
						// Handle questionnaire
						send <- map[string]interface{}{
							"type": "questionnaire",
							"payload": map[string]interface{}{
								"questions": docPayload.Questions,
							},
						}

						// Save AI message to DB with FULL JSON content so we can restore it
						if msgRepo != nil {
							// Serialize questions to JSON
							qBytes, _ := json.Marshal(docPayload.Questions)
							// We use a special prefix to identify this as a questionnaire in history
							storedText := "[QUESTIONNAIRE_JSON]" + string(qBytes)

							am := &models.Message{
								SessionID: p.SessionID,
								Author:    "ai",
								Text:      storedText,
								CreatedAt: time.Now().Unix(),
							}
							_ = msgRepo.Save(am)
						}
						continue // Skip the default ai_done at the bottom
					}
				} else {
					// Not JSON, just normal text
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
				}

			default:
				send <- map[string]interface{}{"type": "error", "payload": map[string]string{"msg": "unknown message type"}}
			}
		}
	}
}
