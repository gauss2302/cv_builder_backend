package domain

import (
	"encoding/json"
	"strings"
	"time"
)

type Education struct {
	Institution string `json:"institution"`
	Location    string `json:"location"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	StartDate   string `json:"start_date"` // Format: YYYY-MM-DD
	EndDate     string `json:"end_date"`   // Format: YYYY-MM-DD or "Present"
	Description string `json:"description"`
}

func (e *Education) Validate() error {
	if strings.TrimSpace(e.Institution) == "" {
		return NewValidationError("institution", "institution is required", ErrInvalidField)
	}
	if strings.TrimSpace(e.Degree) == "" {
		return NewValidationError("degree", "degree is required", ErrInvalidField)
	}
	if strings.TrimSpace(e.StartDate) == "" {
		return NewValidationError("start_date", "start date is required", ErrInvalidField)
	}

	if e.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", e.StartDate)
		if err != nil {
			return NewValidationError("start_date", "Invalid start date format (must be YYYY-MM-DD)", ErrInvalidField)
		}

		if e.EndDate != "" && e.EndDate != "Present" {
			endDate, err := time.Parse("2006-01-02", e.EndDate)
			if err != nil {
				return NewValidationError("end_date", "Invalid end date format (must be YYYY-MM-DD or 'Present')", ErrInvalidField)
			}

			if endDate.Before(startDate) {
				return NewValidationError("end_date", "End date must be after start date", ErrDateRange)
			}
		}
	}
	return nil
}

func (e *Education) BeforeSave() {
	e.Institution = strings.TrimSpace(e.Institution)
	e.Location = strings.TrimSpace(e.Location)
	e.Degree = strings.TrimSpace(e.Degree)
	e.Field = strings.TrimSpace(e.Field)
	e.StartDate = strings.TrimSpace(e.StartDate)
	e.EndDate = strings.TrimSpace(e.EndDate)
	e.Description = strings.TrimSpace(e.Description)
}

func (e *Education) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Education) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}
