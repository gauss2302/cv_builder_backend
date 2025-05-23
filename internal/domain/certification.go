package domain

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type Certification struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	IssueDate    string `json:"issue_date"`            // Format: YYYY-MM-DD
	ExpiryDate   string `json:"expiry_date,omitempty"` // Format: YYYY-MM-DD or "No Expiration"
	CredentialID string `json:"credential_id,omitempty"`
	URL          string `json:"url,omitempty"`
}

func (c *Certification) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return NewValidationError("name", "Certification name is required", ErrInvalidField)
	}
	if strings.TrimSpace(c.Issuer) == "" {
		return NewValidationError("issuer", "Issuer is required", ErrInvalidField)
	}
	if strings.TrimSpace(c.IssueDate) == "" {
		return NewValidationError("issue_date", "Issue date is required", ErrInvalidField)
	}

	// Validate dates
	if c.IssueDate != "" {
		issueDate, err := time.Parse("2006-01-02", c.IssueDate)
		if err != nil {
			return NewValidationError("issue_date", "Invalid issue date format (must be YYYY-MM-DD)", ErrInvalidField)
		}

		if c.ExpiryDate != "" && c.ExpiryDate != "No Expiration" {
			expiryDate, err := time.Parse("2006-01-02", c.ExpiryDate)
			if err != nil {
				return NewValidationError("expiry_date", "Invalid expiry date format (must be YYYY-MM-DD or 'No Expiration')", ErrInvalidField)
			}

			if expiryDate.Before(issueDate) {
				return NewValidationError("expiry_date", "Expiry date must be after issue date", ErrDateRange)
			}
		}
	}

	if c.URL != "" {
		if _, err := url.ParseRequestURI(c.URL); err != nil {
			return NewValidationError("url", "Invalid URL", ErrInvalidField)
		}
	}

	return nil
}

func (c *Certification) BeforeSave() {
	c.Name = strings.TrimSpace(c.Name)
	c.Issuer = strings.TrimSpace(c.Issuer)
	c.IssueDate = strings.TrimSpace(c.IssueDate)
	c.ExpiryDate = strings.TrimSpace(c.ExpiryDate)
	c.CredentialID = strings.TrimSpace(c.CredentialID)
	c.URL = strings.TrimSpace(c.URL)
}

func (c *Certification) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Certification) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}
