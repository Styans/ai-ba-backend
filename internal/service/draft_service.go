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

func (s *DraftService) CreateFromSession(sessionID uint, msgRepo *repository.MessageRepo) (*models.Draft, error) {
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
	}

	if err := s.repo.Create(draft); err != nil {
		return nil, fmt.Errorf("db save failed: %w", err)
	}

	return draft, nil
}

func (s *DraftService) List(limit int) ([]models.Draft, error) {
	return s.repo.List(0, limit)
}

func (s *DraftService) GetByID(id uint) (*models.Draft, error) {
	return s.repo.GetByID(id)
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
