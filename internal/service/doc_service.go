package service

import (
	"fmt"
	"os"
	"time"

	"github.com/gingfrederik/docx"
)

type DocService struct{}

func NewDocService() *DocService {
	return &DocService{}
}

type AnalysisData struct {
	Goal          string      `json:"goal"`
	Description   string      `json:"description"`
	Scope         interface{} `json:"scope"`
	BusinessRules []string    `json:"business_rules"`
	KPIs          []string    `json:"kpis"`
	UseCases      []string    `json:"use_cases"`
	UserStories   []string    `json:"user_stories"`
	DiagramsDesc  []string    `json:"diagrams_desc"` // Description of needed diagrams
}

type QuestionPair struct {
	Step     int    `json:"step"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type SmartRequirements struct {
	Specific   string `json:"specific"`
	Measurable string `json:"measurable"`
	Achievable string `json:"achievable"`
	Relevant   string `json:"relevant"`
	TimeBound  string `json:"time_bound"`
}

type SmartAnalysisData struct {
	Questions         []QuestionPair    `json:"questions"`
	SmartRequirements SmartRequirements `json:"smart_requirements"`
	Summary           string            `json:"summary"`
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
	var scopeText string
	switch v := data.Scope.(type) {
	case string:
		scopeText = v
	case map[string]interface{}:
		if in, ok := v["in_scope"].([]interface{}); ok {
			scopeText += "In Scope:\n"
			for _, i := range in {
				scopeText += fmt.Sprintf("- %v\n", i)
			}
		}
		if out, ok := v["out_of_scope"].([]interface{}); ok {
			scopeText += "\nOut of Scope:\n"
			for _, i := range out {
				scopeText += fmt.Sprintf("- %v\n", i)
			}
		}
	default:
		scopeText = fmt.Sprintf("%v", v)
	}
	s.addSection(f, "3. Scope", scopeText)

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
	if err := os.MkdirAll("storage", 0755); err != nil {
		return "", fmt.Errorf("failed to create storage dir: %w", err)
	}
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
