package dto

type DraftCreateRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type DraftResponse struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
