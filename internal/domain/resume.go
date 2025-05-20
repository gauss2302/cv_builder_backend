package domain

import (
	"context"
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

type ResumeRepository interface {
	CreateCV(ctx context.Context, userId uuid.UUID) (*Resume, error)
	GetCVById(ctx context.Context, id uuid.UUID) (*Resume, error)
	GetCVByUserId(ctx context.Context, userId uuid.UUID) ([]*Resume, error)
	DeleteCV(ctx context.Context, id uuid.UUID) error

	SavePersonalInfo(ctx context.Context, resumeID uuid.UUID, info *PersonalInfo) error
	GetPersonalInfo(ctx context.Context, resumeID uuid.UUID) (*PersonalInfo, error)

	AddEducation(ctx context.Context, resumeID uuid.UUID, education *Education) (uuid.UUID, error)
	UpdateEducation(ctx context.Context, id uuid.UUID, education *Education) error
	DeleteEducation(ctx context.Context, id uuid.UUID) error
	GetEducation(ctx context.Context, id uuid.UUID) (*Education, error)
	GetEducationByResume(ctx context.Context, resumeID uuid.UUID) ([]*Education, error)

	AddExperience(ctx context.Context, resumeId uuid.UUID, experience *Experience) (uuid.UUID, error)
	UpdateExperience(ctx context.Context, id uuid.UUID, experience *Experience) error
	DeleteExperience(ctx context.Context, id uuid.UUID) error
	GetExperience(ctx context.Context, id uuid.UUID) (*Experience, error)
	GetExperienceByResume(ctx context.Context, resumeID uuid.UUID) ([]*Experience, error)

	AddSkill(ctx context.Context, resumeID uuid.UUID, skill *Skill) (uuid.UUID, error)
	UpdateSkill(ctx context.Context, id uuid.UUID, skill *Skill) error
	DeleteSkill(ctx context.Context, id uuid.UUID) error
	GetSkill(ctx context.Context, id uuid.UUID) (*Skill, error)
	GetSkillsByCV(ctx context.Context, resumeID uuid.UUID) ([]*Skill, error)

	// Project operations
	AddProject(ctx context.Context, resumeID uuid.UUID, project *Project) (uuid.UUID, error)
	UpdateProject(ctx context.Context, id uuid.UUID, project *Project) error
	DeleteProject(ctx context.Context, id uuid.UUID) error
	GetProject(ctx context.Context, id uuid.UUID) (*Project, error)
	GetProjectByCV(ctx context.Context, resumeId uuid.UUID) ([]*Project, error)

	// Certification operations
	AddCertification(ctx context.Context, resumeID uuid.UUID, certification *Certification) (uuid.UUID, error)
	UpdateCertification(ctx context.Context, id uuid.UUID, certification *Certification) error
	DeleteCertification(ctx context.Context, id uuid.UUID) error
	GetCertification(ctx context.Context, id uuid.UUID) (*Certification, error)
	GetCertificationsByResume(ctx context.Context, resumeID uuid.UUID) ([]*Certification, error)

	// Complete resume operations
	GetCompleteResume(ctx context.Context, resumeID uuid.UUID) (*Resume, error)
}
