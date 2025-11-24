package dto

type StartChatRequest struct {
	// ...existing code...
	Title string `json:"title"`
}

type SendChatRequest struct {
	SessionID uint   `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	SessionID uint   `json:"session_id"`
	Reply     string `json:"reply"`
}
