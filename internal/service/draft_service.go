package service

import (
	"ai-ba/internal/domain/models"
	"ai-ba/internal/repository"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type DraftService struct {
	repo *repository.DraftRepo
	llm  *LLMService
	doc  *DocService
}

func NewDraftService(repo *repository.DraftRepo, llm *LLMService, doc *DocService) *DraftService {
	return &DraftService{
		repo: repo,
		llm:  llm,
		doc:  doc,
	}
}

func (s *DraftService) CreateFromRequest(title, userReq string) (*models.Draft, error) {
	// 1. Analyze
	analysis, err := s.llm.AnalyzeRequest(userReq)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// 2. Generate Doc
	filename := fmt.Sprintf("draft_%d_%d.docx", time.Now().Unix(), rand.Intn(1000))
	path, err := s.doc.GenerateBADocument(*analysis, filename)
	if err != nil {
		return nil, fmt.Errorf("doc generation failed: %w", err)
	}

	// 3. Save
	jsonBytes, _ := json.Marshal(analysis)
	draft := &models.Draft{
		Title:             title,
		Content:           userReq,
		Status:            "PENDING",
		FilePath:          path,
		StructuredContent: jsonBytes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.Create(draft); err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}

	return draft, nil
}

func (s *DraftService) CreateFromSession(sessionID uint, userID uint, msgRepo *repository.MessageRepo) (*models.Draft, error) {
	// 1. Fetch history
	msgs, err := msgRepo.GetBySessionID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var history []string
	for _, m := range msgs {
		role := "User"
		if m.Author == "ai" {
			role = "Analyst"
		}
		history = append(history, fmt.Sprintf("%s: %s", role, m.Text))
	}

	// 2. Extract Data
	analysis, err := s.llm.ExtractDataFromChat(history)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// 3. Generate Doc
	filename := fmt.Sprintf("draft_session_%d_%d.docx", sessionID, time.Now().Unix())
	path, err := s.doc.GenerateBADocument(*analysis, filename)
	if err != nil {
		return nil, fmt.Errorf("doc generation failed: %w", err)
	}

	// 4. Save Draft
	jsonBytes, _ := json.Marshal(analysis)
	draft := &models.Draft{
		Title:             fmt.Sprintf("Session %d Report", sessionID),
		Content:           "Generated from chat session",
		Status:            "PENDING",
		FilePath:          path,
		StructuredContent: jsonBytes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		SessionID:         sessionID,
		UserID:            userID,
	}

	if err := s.repo.Create(draft); err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}

	return draft, nil
}

func (s *DraftService) List(limit int) ([]models.Draft, error) {
	return s.repo.List(0, limit)
}

func (s *DraftService) ListByUser(userID uint) ([]models.Draft, error) {
	return s.repo.ListByUser(userID)
}

func (s *DraftService) GetByID(id uint) (*models.Draft, error) {
	return s.repo.GetByID(id)
}

func (s *DraftService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *DraftService) DeleteAllByUser(userID uint) error {
	return s.repo.DeleteAllByUser(userID)
}

func (s *DraftService) Approve(id uint) error {
	d, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	d.Status = "APPROVED"
	d.UpdatedAt = time.Now()
	return s.repo.Update(d)
}

func (s *DraftService) CreateFromSmartData(sessionID uint, userID uint, data *SmartAnalysisData) (*models.Draft, error) {
	// 1. Generate Doc
	filename := fmt.Sprintf("draft_session_%d_%d.docx", sessionID, time.Now().Unix())
	path, err := s.doc.GenerateSmartBADocument(*data, filename)
	if err != nil {
		return nil, fmt.Errorf("doc generation failed: %w", err)
	}

	// 2. Save Draft
	jsonBytes, _ := json.Marshal(data)
	draft := &models.Draft{
		Title:             fmt.Sprintf("Session %d Report (SMART)", sessionID),
		Content:           data.Summary,
		Status:            "PENDING",
		FilePath:          path,
		StructuredContent: jsonBytes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		SessionID:         sessionID,
		UserID:            userID,
	}

	if err := s.repo.Create(draft); err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}

	return draft, nil
}

func (s *DraftService) CreateFromAnalysisData(sessionID uint, userID uint, data *AnalysisData) (*models.Draft, error) {
	// 1. Generate Doc
	filename := fmt.Sprintf("draft_session_%d_%d.docx", sessionID, time.Now().Unix())
	path, err := s.doc.GenerateBADocument(*data, filename)
	if err != nil {
		return nil, fmt.Errorf("doc generation failed: %w", err)
	}

	// 2. Check if draft exists for this session
	jsonBytes, _ := json.Marshal(data)
	existingDraft, err := s.repo.GetBySessionID(sessionID)

	if err == nil && existingDraft != nil {
		// Update existing draft
		existingDraft.Title = data.Project.Name
		if existingDraft.Title == "" {
			existingDraft.Title = fmt.Sprintf("Session %d Report (Detailed)", sessionID)
		}
		existingDraft.Content = data.ExecutiveSummary.Goal
		existingDraft.FilePath = path
		existingDraft.StructuredContent = jsonBytes
		existingDraft.UpdatedAt = time.Now()

		if err := s.repo.Update(existingDraft); err != nil {
			return nil, fmt.Errorf("db update failed: %w", err)
		}
		return existingDraft, nil
	}

	// Create new draft
	title := data.Project.Name
	if title == "" {
		title = fmt.Sprintf("Session %d Report (Detailed)", sessionID)
	}

	draft := &models.Draft{
		Title:             title,
		Content:           data.ExecutiveSummary.Goal, // Use goal as short content/summary
		Status:            "PENDING",
		FilePath:          path,
		StructuredContent: jsonBytes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		SessionID:         sessionID,
		UserID:            userID,
	}

	if err := s.repo.Create(draft); err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}

	return draft, nil
}

type BusinessRequestsResponse struct {
	Reviewing []models.Draft `json:"reviewing"`
	Accepted  []models.Draft `json:"accepted"`
	Rejected  []models.Draft `json:"rejected"`
}

func (s *DraftService) GetBusinessRequests(userID uint, role string) (*BusinessRequestsResponse, error) {
	var drafts []models.Draft
	var err error

	// Admin/BA sees all, User sees own
	if role == "Business Analyst" || role == "admin" {
		drafts, err = s.repo.List(0, 1000) // Fetch all (with limit)
	} else {
		drafts, err = s.repo.ListByUser(userID)
	}

	if err != nil {
		return nil, err
	}

	resp := &BusinessRequestsResponse{
		Reviewing: []models.Draft{},
		Accepted:  []models.Draft{},
		Rejected:  []models.Draft{},
	}

	for _, d := range drafts {
		switch d.Status {
		case "PENDING":
			resp.Reviewing = append(resp.Reviewing, d)
		case "APPROVED":
			resp.Accepted = append(resp.Accepted, d)
		case "REJECTED":
			resp.Rejected = append(resp.Rejected, d)
		default:
			// Treat unknown as reviewing/pending for now, or ignore
			resp.Reviewing = append(resp.Reviewing, d)
		}
	}

	return resp, nil
}
