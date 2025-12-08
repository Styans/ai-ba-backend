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

type ProjectInfo struct {
	Name           string `json:"name"`
	Manager        string `json:"manager"`
	DateSubmitted  string `json:"date_submitted"`
	DocumentStatus string `json:"document_status"`
}

type ExecutiveSummary struct {
	ProblemStatement string `json:"problem_statement"`
	Goal             string `json:"goal"`
	ExpectedOutcomes string `json:"expected_outcomes"`
}

type ProjectScope struct {
	InScope    []string `json:"in_scope"`
	OutOfScope []string `json:"out_of_scope"`
}

type BusinessRequirement struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	PriorityLevel string `json:"priority_level"`
	CriticalLevel string `json:"critical_level"`
}

type Stakeholder struct {
	Name    string `json:"name"`
	JobRole string `json:"job_role"`
	Duties  string `json:"duties"`
}

type ProjectConstraint struct {
	Constraint  string `json:"constraint"`
	Description string `json:"description"`
}

type CostBenefitAnalysis struct {
	Costs       []string `json:"costs"`
	Benefits    []string `json:"benefits"`
	TotalCost   string   `json:"total_cost"`
	ExpectedROI string   `json:"expected_roi"`
}

type FunctionalRequirement struct {
	Module   string   `json:"module"`
	Features []string `json:"features"`
}

type NonFunctionalRequirements struct {
	Performance    string   `json:"performance"`
	Security       []string `json:"security"`
	Availability   string   `json:"availability"`
	Scalability    string   `json:"scalability"`
	UXRequirements string   `json:"ux_requirements"`
}

type UIUXStyleGuide struct {
	Colors     map[string]interface{} `json:"colors"`
	Typography map[string]interface{} `json:"typography"`
	Components map[string]interface{} `json:"components"`
}

type FrontendStyles struct {
	Layout     map[string]interface{} `json:"layout"`
	Animations map[string]interface{} `json:"animations"`
}

type UseCase struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Actors           []string `json:"actors"`
	PreConditions    string   `json:"pre_conditions"`
	PostConditions   string   `json:"post_conditions"`
	MainFlow         []string `json:"main_flow"`
	AlternativeFlows []string `json:"alternative_flows"`
}

type UserStory struct {
	ID                 string   `json:"id"`
	Role               string   `json:"role"`
	Action             string   `json:"action"`
	Benefit            string   `json:"benefit"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type ProcessFlow struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MermaidCode string `json:"mermaid_code"` // Mermaid JS syntax for diagram
}

type AnalysisData struct {
	Project                   ProjectInfo               `json:"project"`
	ExecutiveSummary          ExecutiveSummary          `json:"executive_summary"`
	ProjectObjectives         []string                  `json:"project_objectives"`
	ProjectScope              ProjectScope              `json:"project_scope"`
	BusinessRequirements      []BusinessRequirement     `json:"business_requirements"`
	KeyStakeholders           []Stakeholder             `json:"key_stakeholders"`
	ProjectConstraints        []ProjectConstraint       `json:"project_constraints"`
	CostBenefitAnalysis       CostBenefitAnalysis       `json:"cost_benefit_analysis"`
	FunctionalRequirements    []FunctionalRequirement   `json:"functional_requirements"`
	NonFunctionalRequirements NonFunctionalRequirements `json:"non_functional_requirements"`
	UseCases                  []UseCase                 `json:"use_cases"`
	UserStories               []UserStory               `json:"user_stories"`
	ProcessFlows              []ProcessFlow             `json:"process_flows"`
	LeadingIndicators         []string                  `json:"leading_indicators"`
	UIUXStyleGuide            UIUXStyleGuide            `json:"ui_ux_style_guide"`
	FrontendStyles            FrontendStyles            `json:"frontend_styles"`
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
	Answers           []map[string]interface{} `json:"answers"` // Flexible to handle various formats
	SmartRequirements SmartRequirements        `json:"smart_requirements"`
	Summary           string                   `json:"summary"`
	Confluence        struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	} `json:"confluence"`
}

func (s *DocService) GenerateBADocument(data AnalysisData, filename string) (string, error) {
	f := docx.NewFile()

	// Title
	p := f.AddParagraph()
	p.AddText("Business Requirements Document (BRD)").Size(24)

	// Meta
	f.AddParagraph().AddText(fmt.Sprintf("Generated on: %s", time.Now().Format(time.RFC1123)))

	// Project Info
	s.addSection(f, "Project Information", fmt.Sprintf("Project Name: %s\nManager: %s\nDate Submitted: %s\nStatus: %s", data.Project.Name, data.Project.Manager, data.Project.DateSubmitted, data.Project.DocumentStatus))

	// Executive Summary
	s.addSection(f, "1. Executive Summary", fmt.Sprintf("Problem Statement: %s\n\nGoal: %s\n\nExpected Outcomes: %s", data.ExecutiveSummary.ProblemStatement, data.ExecutiveSummary.Goal, data.ExecutiveSummary.ExpectedOutcomes))

	// Project Objectives
	s.addListSection(f, "2. Project Objectives", data.ProjectObjectives)

	// Project Scope
	f.AddParagraph().AddText("3. Project Scope").Size(16)
	s.addSubSection(f, "In Scope", data.ProjectScope.InScope)
	s.addSubSection(f, "Out of Scope", data.ProjectScope.OutOfScope)

	// Business Requirements
	f.AddParagraph().AddText("4. Business Requirements").Size(16)
	for _, br := range data.BusinessRequirements {
		f.AddParagraph().AddText(fmt.Sprintf("%s: %s (Priority: %s, Critical: %s)", br.ID, br.Description, br.PriorityLevel, br.CriticalLevel))
	}
	f.AddParagraph()

	// Key Stakeholders
	f.AddParagraph().AddText("5. Key Stakeholders").Size(16)
	for _, st := range data.KeyStakeholders {
		f.AddParagraph().AddText(fmt.Sprintf("Name: %s, Role: %s\nDuties: %s", st.Name, st.JobRole, st.Duties))
		f.AddParagraph()
	}

	// Project Constraints
	f.AddParagraph().AddText("6. Project Constraints").Size(16)
	for _, c := range data.ProjectConstraints {
		f.AddParagraph().AddText(fmt.Sprintf("%s: %s", c.Constraint, c.Description))
	}
	f.AddParagraph()

	// Cost Benefit Analysis
	f.AddParagraph().AddText("7. Cost Benefit Analysis").Size(16)
	s.addSubSection(f, "Costs", data.CostBenefitAnalysis.Costs)
	s.addSubSection(f, "Benefits", data.CostBenefitAnalysis.Benefits)
	f.AddParagraph().AddText(fmt.Sprintf("Total Cost: %s", data.CostBenefitAnalysis.TotalCost))
	f.AddParagraph().AddText(fmt.Sprintf("Expected ROI: %s", data.CostBenefitAnalysis.ExpectedROI))
	f.AddParagraph()

	// Functional Requirements
	f.AddParagraph().AddText("8. Functional Requirements").Size(16)
	for _, fr := range data.FunctionalRequirements {
		s.addSubSection(f, fmt.Sprintf("Module: %s", fr.Module), fr.Features)
	}

	// Non-Functional Requirements
	f.AddParagraph().AddText("9. Non-Functional Requirements").Size(16)
	f.AddParagraph().AddText("Performance: " + data.NonFunctionalRequirements.Performance)
	s.addSubSection(f, "Security", data.NonFunctionalRequirements.Security)
	f.AddParagraph().AddText("Availability: " + data.NonFunctionalRequirements.Availability)
	f.AddParagraph().AddText("Scalability: " + data.NonFunctionalRequirements.Scalability)
	f.AddParagraph().AddText("UX Requirements: " + data.NonFunctionalRequirements.UXRequirements)
	f.AddParagraph()

	// 10. Analytical Artifacts
	f.AddParagraph().AddText("10. Analytical Artifacts").Size(16)

	// User Stories
	f.AddParagraph().AddText("10.1 User Stories").Size(14)
	for _, us := range data.UserStories {
		f.AddParagraph().AddText(fmt.Sprintf("%s: As a %s, I want to %s, so that %s", us.ID, us.Role, us.Action, us.Benefit))
		s.addSubSection(f, "Acceptance Criteria", us.AcceptanceCriteria)
	}

	// Use Cases
	f.AddParagraph().AddText("10.2 Use Cases").Size(14)
	for _, uc := range data.UseCases {
		f.AddParagraph().AddText(fmt.Sprintf("%s: %s", uc.ID, uc.Name)).Size(12)
		f.AddParagraph().AddText(uc.Description)
		f.AddParagraph().AddText("Actors: " + fmt.Sprintf("%v", uc.Actors))
		f.AddParagraph().AddText("Pre-conditions: " + uc.PreConditions)
		s.addSubSection(f, "Main Flow", uc.MainFlow)
		s.addSubSection(f, "Alternative Flows", uc.AlternativeFlows)
		f.AddParagraph().AddText("Post-conditions: " + uc.PostConditions)
		f.AddParagraph()
	}

	// Process Flows
	f.AddParagraph().AddText("10.3 Process Flows").Size(14)
	for _, pf := range data.ProcessFlows {
		f.AddParagraph().AddText(pf.Name)
		f.AddParagraph().AddText(pf.Description)
		f.AddParagraph().AddText("Mermaid Diagram Code:")
		f.AddParagraph().AddText(pf.MermaidCode).Size(10) // Render code as text for now
		f.AddParagraph()
	}

	// Leading Indicators
	s.addListSection(f, "10.4 Leading Indicators (KPIs)", data.LeadingIndicators)

	// UI/UX Style Guide
	f.AddParagraph().AddText("10. UI/UX Style Guide").Size(16)
	f.AddParagraph().AddText("Colors:")
	for k, v := range data.UIUXStyleGuide.Colors {
		f.AddParagraph().AddText(fmt.Sprintf("- %s: %s", k, v))
	}
	f.AddParagraph().AddText("Typography:")
	for k, v := range data.UIUXStyleGuide.Typography {
		f.AddParagraph().AddText(fmt.Sprintf("- %s: %s", k, v))
	}
	f.AddParagraph().AddText("Components:")
	for k, v := range data.UIUXStyleGuide.Components {
		f.AddParagraph().AddText(fmt.Sprintf("- %s: %s", k, v))
	}
	f.AddParagraph()

	// Frontend Styles
	f.AddParagraph().AddText("11. Frontend Styles").Size(16)
	f.AddParagraph().AddText("Layout:")
	for k, v := range data.FrontendStyles.Layout {
		f.AddParagraph().AddText(fmt.Sprintf("- %s: %s", k, v))
	}
	f.AddParagraph().AddText("Animations:")
	for k, v := range data.FrontendStyles.Animations {
		f.AddParagraph().AddText(fmt.Sprintf("- %s: %s", k, v))
	}
	f.AddParagraph()

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

func (s *DocService) GenerateSmartBADocument(data SmartAnalysisData, filename string) (string, error) {
	f := docx.NewFile()

	// Title
	p := f.AddParagraph()
	p.AddText("Business Analysis Report (SMART)").Size(24)

	// Meta
	f.AddParagraph().AddText(fmt.Sprintf("Generated on: %s", time.Now().Format(time.RFC1123)))

	// Summary
	s.addSection(f, "Executive Summary", data.Summary)

	// SMART Requirements
	f.AddParagraph().AddText("SMART Requirements").Size(16)
	f.AddParagraph().AddText("Specific: " + data.SmartRequirements.Specific)
	f.AddParagraph().AddText("Measurable: " + data.SmartRequirements.Measurable)
	f.AddParagraph().AddText("Achievable: " + data.SmartRequirements.Achievable)
	f.AddParagraph().AddText("Relevant: " + data.SmartRequirements.Relevant)
	f.AddParagraph().AddText("Time-bound: " + data.SmartRequirements.TimeBound)
	f.AddParagraph()

	// Q&A History
	f.AddParagraph().AddText("Interview Transcript").Size(16)
	for i, ans := range data.Answers {
		qText := ""
		if v, ok := ans["question"].(string); ok {
			qText = v
		} else if v, ok := ans["question_id"].(string); ok {
			qText = fmt.Sprintf("Question ID: %s", v)
		} else {
			qText = fmt.Sprintf("Question #%d", i+1)
		}

		aText := ""
		if v, ok := ans["answer"].(string); ok {
			aText = v
		}

		f.AddParagraph().AddText(fmt.Sprintf("Q: %s", qText)).Size(12)
		f.AddParagraph().AddText("A: " + aText)
		f.AddParagraph()
	}

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
