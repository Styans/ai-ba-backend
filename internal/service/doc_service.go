package service

import (
	"fmt"
	"time"

	"github.com/gingfrederik/docx"
)

type DocService struct{}

func NewDocService() *DocService {
	return &DocService{}
}

type AnalysisData struct {
	Goal          string   `json:"goal"`
	Description   string   `json:"description"`
	Scope         string   `json:"scope"`
	BusinessRules []string `json:"business_rules"`
	KPIs          []string `json:"kpis"`
	UseCases      []string `json:"use_cases"`
	UserStories   []string `json:"user_stories"`
	DiagramsDesc  []string `json:"diagrams_desc"` // Description of needed diagrams
}

func (s *DocService) GenerateBADocument(data AnalysisData, filename string) (string, error) {
	f := docx.NewFile()

	// Title
	p := f.AddParagraph()
	p.AddText("Business Analysis Report").Size(24)

	// Meta
	f.AddParagraph().AddText(fmt.Sprintf("Generated on: %s", time.Now().Format(time.RFC1123)))

	// 1. Goal
	s.addSection(f, "1. Goal", data.Goal)

	// 2. Description
	s.addSection(f, "2. Description", data.Description)

	// 3. Scope
	s.addSection(f, "3. Scope", data.Scope)

	// 4. Business Rules
	s.addListSection(f, "4. Business Rules", data.BusinessRules)

	// 5. KPIs
	s.addListSection(f, "5. KPIs", data.KPIs)

	// 6. Analytical Artifacts
	f.AddParagraph().AddText("6. Analytical Artifacts").Size(16)

	s.addSubSection(f, "6.1 Use Cases", data.UseCases)
	s.addSubSection(f, "6.2 User Stories", data.UserStories)
	s.addSubSection(f, "6.3 Process Diagrams (Descriptions)", data.DiagramsDesc)

	// Save
	path := fmt.Sprintf("./storage/%s", filename)
	if err := f.Save(path); err != nil {
		return "", err
	}

	return path, nil
}

func (s *DocService) addSection(f *docx.File, title, content string) {
	f.AddParagraph().AddText(title).Size(16)
	f.AddParagraph().AddText(content)
	f.AddParagraph() // spacer
}

func (s *DocService) addListSection(f *docx.File, title string, items []string) {
	f.AddParagraph().AddText(title).Size(16)
	for _, item := range items {
		f.AddParagraph().AddText("• " + item)
	}
	f.AddParagraph() // spacer
}

func (s *DocService) addSubSection(f *docx.File, title string, items []string) {
	f.AddParagraph().AddText(title).Size(14)
	for _, item := range items {
		f.AddParagraph().AddText("- " + item)
	}
	f.AddParagraph()
}
