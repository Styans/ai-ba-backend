package service

type DraftService struct{}

func NewDraftService() *DraftService { return &DraftService{} }

func (s *DraftService) Create(title, content string) (uint, error) {
	// TODO
	return 0, nil
}
