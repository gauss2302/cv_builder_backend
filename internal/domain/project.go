package domain

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type Project struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies,omitempty"`
	RepoURL      string   `json:"repo_url,omitempty"`
	DemoURL      string   `json:"demo_url,omitempty"`
	StartDate    string   `json:"start_date,omitempty"` // Format: YYYY-MM-DD
	EndDate      string   `json:"end_date,omitempty"`   // Format: YYYY-MM-DD or "Present"
}

func (p *Project) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return NewValidationError("name", "Project name is required", ErrInvalidField)
	}

	if p.RepoURL != "" {
		if _, err := url.ParseRequestURI(p.RepoURL); err != nil {
			return NewValidationError("repo_url", "Invalid repository URL", ErrInvalidField)
		}
	}

	if p.DemoURL != "" {
		if _, err := url.ParseRequestURI(p.DemoURL); err != nil {
			return NewValidationError("demo_url", "Invalid demo URL", ErrInvalidField)
		}
	}

	if p.StartDate != "" && p.StartDate != "Present" {
		if _, err := time.Parse("2006-01-02", p.StartDate); err != nil {
			return NewValidationError("start_date", "Invalid start date format (must be YYYY-MM-DD)", ErrInvalidField)
		}
	}

	if p.EndDate != "" && p.EndDate != "Present" {
		if _, err := time.Parse("2006-01-02", p.EndDate); err != nil {
			return NewValidationError("end_date", "Invalid end date format (must be YYYY-MM-DD or 'Present')", ErrInvalidField)
		}
	}

	if p.StartDate != "" && p.EndDate != "" && p.EndDate != "Present" {
		startDate, _ := time.Parse("2006-01-02", p.StartDate)
		endDate, _ := time.Parse("2006-01-02", p.EndDate)
		if endDate.Before(startDate) {
			return NewValidationError("end_date", "End date must be after start date", ErrDateRange)
		}
	}

	return nil
}

func (p *Project) BeforeSave() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.RepoURL = strings.TrimSpace(p.RepoURL)
	p.DemoURL = strings.TrimSpace(p.DemoURL)
	p.StartDate = strings.TrimSpace(p.StartDate)
	p.EndDate = strings.TrimSpace(p.EndDate)

	for i, tech := range p.Technologies {
		p.Technologies[i] = strings.TrimSpace(tech)
	}

	filteredTechnologies := make([]string, 0, len(p.Technologies))
	for _, tech := range p.Technologies {
		if tech != "" {
			filteredTechnologies = append(filteredTechnologies, tech)
		}
	}
	p.Technologies = filteredTechnologies
}

func (p *Project) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Project) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}
