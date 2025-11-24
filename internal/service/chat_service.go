package service

type ChatService struct{}

func NewChatService() *ChatService { return &ChatService{} }

func (s *ChatService) StartSession(title string) (uint, error) {
	// TODO
	return 0, nil
}

func (s *ChatService) SendMessage(sessionID uint, message string) (string, error) {
	// TODO: вызвать llm_service
	return "ai reply (placeholder)", nil
}
