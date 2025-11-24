package service

type LLMService struct{}

func NewLLMService() *LLMService { return &LLMService{} }

func (s *LLMService) Generate(prompt string) (string, error) {
	// TODO: вызвать OpenAI / другой LLM
	return "generated text (placeholder)", nil
}
