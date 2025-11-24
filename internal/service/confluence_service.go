package service

type ConfluenceService struct{}

func NewConfluenceService() *ConfluenceService { return &ConfluenceService{} }

func (s *ConfluenceService) Publish(pageTitle, content string) (string, error) {
	// TODO: интеграция
	return "", nil
}
