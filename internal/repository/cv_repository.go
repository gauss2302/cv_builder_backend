package repository

import (
	"cv_builder/internal/domain"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"time"
)

type PostgresCVRepository struct {
	db *sqlx.DB
}

func NewPostgresCVRepository(db *sqlx.DB) *PostgresCVRepository {
	return &PostgresCVRepository{
		db: db,
	}
}

func (r *PostgresCVRepository) CreateCV(userId uuid.UUID) (*domain.Resume, error) {
	query := `
		INSERT INTO resumes (id, user_id, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	resumeID := uuid.New()
	now := time.Now()

	var id uuid.UUID
	err := r.db.QueryRow(
		query,
		resumeID,
		userId,
		now,
	).Scan(&id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create resume")
		return nil, err
	}

	resume := &domain.Resume{
		ID:        resumeID,
		UserID:    userId,
		CreatedAt: now,
	}

	return resume, nil
}

func (r *PostgresCVRepository) GetCVById(id uuid.UUID) (*domain.Resume, error) {
	query := `
		SELECT id, user_id, created_at
		FROM resumes
		WHERE id = $1
	`

	var resume domain.Resume
	err := r.db.Get(&resume, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("resume_id", id.String()).Msg("Failed to get resume by ID")
		return nil, err
	}

	return &resume, nil
}

func (r *PostgresCVRepository) GetCVByUserId(userId uuid.UUID) ([]*domain.Resume, error) {
	query := `
		SELECT id, user_id, created_at
		FROM resumes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var resumes []*domain.Resume
	err := r.db.Select(&resumes, query, userId)
	if err != nil {
		log.Error().Err(err).Str("user_id", userId.String()).Msg("Failed to get resumes by user ID")
		return nil, err
	}

	return resumes, nil
}

func (r *PostgresCVRepository) DeleteCV(id uuid.UUID) error {
	query := `
		DELETE FROM resumes
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Str("resume_id", id.String()).Msg("Failed to delete resume")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get rows affected")
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresCVRepository) SavePersonalInfo(resumeID uuid.UUID, info *domain.PersonalInfo) error {
	query := `
		INSERT INTO personal_info (
			id, resume_id, first_name, last_name, email, phone, 
			street, city, country, job_title, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (resume_id) DO UPDATE SET
			first_name = $3,
			last_name = $4,
			email = $5,
			phone = $6,
			street = $7,
			city = $8,
			country = $9,
			job_title = $10,
			updated_at = $12
		RETURNING id
	`

	// Apply BeforeSave to sanitize the data
	info.BeforeSave()

	id := uuid.New()
	now := time.Now()

	var returnedID uuid.UUID
	err := r.db.QueryRow(
		query,
		id,
		resumeID,
		info.FirstName,
		info.LastName,
		info.Email,
		info.Phone,
		info.Address.Street,
		info.Address.City,
		info.Address.Country,
		info.JobTitle,
		now,
		now,
	).Scan(&returnedID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save personal info")
		return err
	}

	return nil
}

func (r *PostgresCVRepository) GetPersonalInfo(resumeId uuid.UUID) (*domain.PersonalInfo, error) {
	query := `
		SELECT first_name, last_name, email, phone, street, city, country, job_title
		FROM personal_info
		WHERE resume_id = $1
	`

	var info struct {
		FirstName string `db:"first_name"`
		LastName  string `db:"last_name"`
		Email     string `db:"email"`
		Phone     string `db:"phone"`
		Street    string `db:"street"`
		City      string `db:"city"`
		Country   string `db:"country"`
		JobTitle  string `db:"job_title"`
	}

	err := r.db.Get(&info, query, resumeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("failed to get personal info")
		return nil, err
	}

	result := &domain.PersonalInfo{
		FirstName: info.FirstName,
		LastName:  info.LastName,
		Email:     info.Email,
		Phone:     info.Phone,
		JobTitle:  info.JobTitle,
	}

	result.Address.Street = info.Street
	result.Address.City = info.City
	result.Address.Country = info.Country

	return result, nil
}

func (r *PostgresCVRepository) AddEducation(resumeId uuid.UUID, education *domain.Education) (uuid.UUID, error) {
	query := `
		INSERT INTO education (
			id, resume_id, institution, location, degree, field, 
			start_date, end_date, description, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	education.BeforeSave()
	if err := education.Validate(); err != nil {
		return uuid.New(), err
	}

	id := uuid.New()
	now := time.Now()

	var startDate *time.Time
	var endDate *time.Time

	if education.StartDate != "" {
		parsedStartDate, err := time.Parse("2006-01-02", education.StartDate)
		if err != nil {
			return uuid.Nil, err
		}
		startDate = &parsedStartDate
	}

	if education.EndDate != "" && education.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", education.EndDate)
		if err != nil {
			return uuid.Nil, err
		}
		endDate = &parsedEndDate
	}

	var returnedId uuid.UUID
	err := r.db.QueryRow(
		query,
		id,
		resumeId,
		education.Institution,
		education.Location,
		education.Degree,
		education.Field,
		startDate,
		endDate,
		education.Description,
		now,
		now,
	).Scan(&returnedId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add education")
		return uuid.Nil, err
	}

	return returnedId, nil
}

func (r *PostgresCVRepository) UpdateEducation(id uuid.UUID, education *domain.Education) error {
	query := `
		UPDATE education
		SET institution = $1,
			location = $2,
			degree = $3,
			field = $4,
			start_date = $5,
			end_date = $6,
			description = $7,
			updated_at = $8
		WHERE id = $9
	`

	education.BeforeSave()

	if err := education.Validate(); err != nil {
		return err
	}

	now := time.Now()

	var startDate *time.Time
	var endDate *time.Time

	if education.StartDate != "" {
		parsedStartDate, err := time.Parse("2006-01-02", education.StartDate)
		if err != nil {
			return err
		}
		startDate = &parsedStartDate
	}

	if education.EndDate != "" && education.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", education.EndDate)
		if err != nil {
			return err
		}
		endDate = &parsedEndDate
	}

	result, err := r.db.Exec(
		query,
		education.Institution,
		education.Location,
		education.Degree,
		education.Field,
		startDate,
		endDate,
		education.Description,
		now,
		id)

	if err != nil {
		log.Error().Err(err).Msg("failed to update education")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get rows affected")
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresCVRepository) DeleteEducation(id uuid.UUID) error {
	query := `
		DELETE FROM education
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete education")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresCVRepository) GetEducation(id uuid.UUID) (*domain.Education, error) {
	query := `
		SELECT institution, location, degree, field, 
		       start_date, end_date, description
		FROM education
		WHERE id = $1
	`

	var edu struct {
		Institution string     `db:"institution"`
		Location    string     `db:"location"`
		Degree      string     `db:"degree"`
		Field       string     `db:"field"`
		StartDate   time.Time  `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
		Description string     `db:"description"`
	}

	err := r.db.Get(&edu, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("education_id", id.String()).Msg("Failed to get education")
		return nil, err
	}

	// Format dates
	startDate := edu.StartDate.Format("2006-01-02")
	var endDate string
	if edu.EndDate != nil {
		endDate = edu.EndDate.Format("2006-01-02")
	} else {
		endDate = "Present"
	}

	education := &domain.Education{
		Institution: edu.Institution,
		Location:    edu.Location,
		Degree:      edu.Degree,
		Field:       edu.Field,
		StartDate:   startDate,
		EndDate:     endDate,
		Description: edu.Description,
	}

	return education, nil
}

func (r *PostgresCVRepository) GetEducationByResume(resumeID uuid.UUID) ([]*domain.Education, error) {
	query := `
		SELECT id, institution, location, degree, field, 
		       start_date, end_date, description
		FROM education
		WHERE resume_id = $1
		ORDER BY start_date DESC
	`

	type educationRow struct {
		ID          uuid.UUID  `db:"id"`
		Institution string     `db:"institution"`
		Location    string     `db:"location"`
		Degree      string     `db:"degree"`
		Field       string     `db:"field"`
		StartDate   time.Time  `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
		Description string     `db:"description"`
	}

	var rows []educationRow
	err := r.db.Select(&rows, query, resumeID)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeID.String()).Msg("Failed to get education by resume")
		return nil, err
	}

	education := make([]*domain.Education, len(rows))
	for i, row := range rows {
		// Format dates
		startDate := row.StartDate.Format("2006-01-02")
		var endDate string
		if row.EndDate != nil {
			endDate = row.EndDate.Format("2006-01-02")
		} else {
			endDate = "Present"
		}

		education[i] = &domain.Education{
			Institution: row.Institution,
			Location:    row.Location,
			Degree:      row.Degree,
			Field:       row.Field,
			StartDate:   startDate,
			EndDate:     endDate,
			Description: row.Description,
		}
	}

	return education, nil
}
