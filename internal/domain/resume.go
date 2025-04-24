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

type ResumeRepository interface {
	CreateCV(userId uuid.UUID) (*Resume, error)
	GetCVById(id uuid.UUID) (*Resume, error)
	GetCVByUserId(userId uuid.UUID) ([]*Resume, error)
	DeleteCV(id uuid.UUID) error

	SavePersonalInfo(resumeID uuid.UUID, info *PersonalInfo) error
	GetPersonalInfo(resumeID uuid.UUID) (*PersonalInfo, error)

	AddEducation(resumeID uuid.UUID, education *Education) (uuid.UUID, error)
	UpdateEducation(id uuid.UUID, education *Education) error
	DeleteEducation(id uuid.UUID) error
	GetEducation(id uuid.UUID) (*Education, error)
	GetEducationByResume(resumeID uuid.UUID) ([]*Education, error)

	AddExperience(resumeId uuid.UUID, experience *Experience) (uuid.UUID, error)
	UpdateExperience(id uuid.UUID, experience *Experience) error
	DeleteExperience(id uuid.UUID) error
	GetExperience(id uuid.UUID) (*Experience, error)
	GetExperienceByResume(resumeID uuid.UUID) ([]*Experience, error)

	//AddSkill(resumeID uuid.UUID, skill *Skill) (uuid.UUID, error)
	//UpdateSkill(id uuid.UUID, skill *Skill) error
	//DeleteSkill(id uuid.UUID) error
	//GetSkill(id uuid.UUID) (*Skill, error)
	//GetSkillsByResume(resumeID uuid.UUID) ([]*Skill, error)
}
