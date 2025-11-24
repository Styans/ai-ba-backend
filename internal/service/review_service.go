package service

type ReviewService struct{}

func NewReviewService() *ReviewService { return &ReviewService{} }

func (s *ReviewService) SubmitReview(draftID uint, notes string) error {
	// TODO
	return nil
}
