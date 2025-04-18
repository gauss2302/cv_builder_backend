package domain

import (
	"github.com/google/uuid"
	"time"
)

type Resume struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	PersonalInfo   *PersonalInfo    `json:"personal_info,omitempty" db:"-"`
	Education      []*Education     `json:"education,omitempty" db:"-"`
	Experience     []*Experience    `json:"experience,omitempty" db:"-"`
	Skills         []*Skill         `json:"skills,omitempty" db:"-"`
	Projects       []*Project       `json:"projects,omitempty" db:"-"`
	Certifications []*Certification `json:"certifications,omitempty" db:"-"`
}
