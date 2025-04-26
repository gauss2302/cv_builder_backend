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

func (r *PostgresCVRepository) AddSkill(resumeId uuid.UUID, skill *domain.Skill) (uuid.UUID, error) {
	query := `
		INSERT INTO skills (
			id, resume_id, name, category, proficiency, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	skill.BeforeSave()

	if err := skill.Validate(); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	now := time.Now()

	var proficiency any
	if skill.Proficiency != 0 {
		proficiency = skill.Proficiency
	} else {
		proficiency = nil
	}

	var returnedID uuid.UUID
	err := r.db.QueryRow(
		query,
		id,
		resumeId,
		skill.Name,
		skill.Category,
		proficiency,
		now,
		now,
	).Scan(&returnedID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add skill")
		return uuid.Nil, err
	}

	return returnedID, nil

}

func (r *PostgresCVRepository) UpdateSkill(id uuid.UUID, skill *domain.Skill) error {
	query := `
		UPDATE skills
		SET name = $1,
			category = $2,
			proficiency = $3,
			updated_at = $4
		WHERE id = $5
	`

	// Apply BeforeSave to sanitize the data
	skill.BeforeSave()

	// Validate the entry
	if err := skill.Validate(); err != nil {
		return err
	}

	now := time.Now()

	// Use NULL for zero proficiency
	var proficiency any
	if skill.Proficiency != 0 {
		proficiency = skill.Proficiency
	} else {
		proficiency = nil
	}

	result, err := r.db.Exec(
		query,
		skill.Name,
		skill.Category,
		proficiency,
		now,
		id,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to update skill")
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

func (r *PostgresCVRepository) DeleteSkill(id uuid.UUID) error {
	query := `
		DELETE FROM skills
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete skill")
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

func (r *PostgresCVRepository) GetSkill(id uuid.UUID) (*domain.Skill, error) {
	query := `
		SELECT name, category, proficiency
		FROM skills
		WHERE id = $1
	`

	var skillRow struct {
		Name        string `db:"name"`
		Category    string `db:"category"`
		Proficiency *int   `db:"proficiency"`
	}

	err := r.db.Get(&skillRow, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("skill_id", id.String()).Msg("failed to get skill")
		return nil, err
	}

	skill := &domain.Skill{
		Name:     skillRow.Name,
		Category: skillRow.Category,
	}

	if skillRow.Proficiency != nil {
		skill.Proficiency = *skillRow.Proficiency
	}

	return skill, nil
}

func (r *PostgresCVRepository) GetSkillsByCV(resumeId uuid.UUID) ([]*domain.Skill, error) {
	query := `
		SELECT id, name, category, proficiency
		FROM skills
		WHERE resume_id = $1
		ORDER BY category, name
	`

	type skillRow struct {
		ID          uuid.UUID `db:"id"`
		Name        string    `db:"name"`
		Category    string    `db:"category"`
		Proficiency *int      `db:"proficiency"`
	}

	var rows []skillRow
	err := r.db.Select(&rows, query, resumeId)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("failed to get skills by resume")
		return nil, err
	}

	skills := make([]*domain.Skill, len(rows))

	for i, row := range rows {
		skills[i] = &domain.Skill{Name: row.Name, Category: row.Category}
		if row.Proficiency != nil {
			skills[i].Proficiency = *row.Proficiency
		}
	}
	return skills, nil

}

func (r *PostgresCVRepository) AddProject(resumeId uuid.UUID, project *domain.Project) (uuid.UUID, error) {
	query := `
		INSERT INTO projects (
			id, resume_id, name, description, repo_url, demo_url, 
			start_date, end_date, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	project.BeforeSave()

	if err := project.Validate(); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	now := time.Now()

	var startDate *time.Time
	var endDate *time.Time

	if project.StartDate != "" && project.StartDate != "Present" {
		parsedStartDate, err := time.Parse("2006-01-02", project.StartDate)
		if err != nil {
			return uuid.Nil, err
		}
		startDate = &parsedStartDate
	}

	if project.EndDate != "" && project.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", project.EndDate)
		if err != nil {
			return uuid.Nil, err
		}
		endDate = &parsedEndDate
	}

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error().Err(err).Msg("failed to begin transaction")
		return uuid.Nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var returnedId uuid.UUID
	err = tx.QueryRow(
		query,
		id,
		resumeId,
		project.Name,
		project.Description,
		project.RepoURL,
		project.DemoURL,
		startDate,
		endDate,
		now,
		now,
	).Scan(&returnedId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add project")
		return uuid.Nil, err
	}

	for _, tech := range project.Technologies {
		err := r.addProjectTechnology(tx, id, tech)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add project technology")
			return uuid.Nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return uuid.Nil, err
	}

	return returnedId, nil
}

// helper
func (r *PostgresCVRepository) addProjectTechnology(tx *sqlx.Tx, projectId uuid.UUID, technology string) error {
	query := `
		INSERT INTO project_technologies (id, project_id, technology)
		VALUES ($1, $2, $3)
	`

	id := uuid.New()
	_, err := tx.Exec(query, id, projectId, technology)
	return err
}

func (r *PostgresCVRepository) UpdateProject(id uuid.UUID, project *domain.Project) error {
	query := `
		UPDATE projects
		SET name = $1,
			description = $2,
			repo_url = $3,
			demo_url = $4,
			start_date = $5,
			end_date = $6,
			updated_at = $7
		WHERE id = $8
	`

	// Apply BeforeSave to sanitize the data
	project.BeforeSave()

	// Validate the entry
	if err := project.Validate(); err != nil {
		return err
	}

	now := time.Now()

	// Parse dates
	var startDate *time.Time
	var endDate *time.Time

	if project.StartDate != "" && project.StartDate != "Present" {
		parsedStartDate, err := time.Parse("2006-01-02", project.StartDate)
		if err != nil {
			return err
		}
		startDate = &parsedStartDate
	}

	if project.EndDate != "" && project.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", project.EndDate)
		if err != nil {
			return err
		}
		endDate = &parsedEndDate
	}

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error().Err(err).Msg("Failed to begin transaction")
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	result, err := tx.Exec(
		query,
		project.Name,
		project.Description,
		project.RepoURL,
		project.DemoURL,
		startDate,
		endDate,
		now,
		id,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update project")
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

	// Delete existing technologies
	_, err = tx.Exec("DELETE FROM project_technologies WHERE project_id = $1", id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete project technologies")
		return err
	}

	// Add updated technologies
	for _, tech := range project.Technologies {
		err = r.addProjectTechnology(tx, id, tech)
		if err != nil {
			log.Error().Err(err).Msg("failed to add project technology")
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return err
	}

	return nil
}

func (r *PostgresCVRepository) DeleteProject(id uuid.UUID) error {
	query := `
		DELETE FROM projects
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete project")
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

func (r *PostgresCVRepository) GetProject(id uuid.UUID) (*domain.Project, error) {
	query := `
		SELECT name, description, repo_url, demo_url, start_date, end_date
		FROM projects
		WHERE id = $1
	`

	var projectRow struct {
		Name        string     `db:"name"`
		Description string     `db:"description"`
		RepoURL     string     `db:"repo_url"`
		DemoURL     string     `db:"demo_url"`
		StartDate   *time.Time `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
	}

	err := r.db.Get(&projectRow, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("project_id", id.String()).Msg("Failed to get project")
		return nil, err
	}

	// Format dates
	var startDate, endDate string
	if projectRow.StartDate != nil {
		startDate = projectRow.StartDate.Format("2006-01-02")
	}
	if projectRow.EndDate != nil {
		endDate = projectRow.EndDate.Format("2006-01-02")
	} else {
		endDate = "Present"
	}

	// Get technologies
	technologies, err := r.GetProjectTechnologies(id)
	if err != nil {
		log.Error().Err(err).Str("project_id", id.String()).Msg("failed to get project technologies")
		return nil, err
	}

	project := &domain.Project{
		Name:         projectRow.Name,
		Description:  projectRow.Description,
		RepoURL:      projectRow.RepoURL,
		DemoURL:      projectRow.DemoURL,
		StartDate:    startDate,
		EndDate:      endDate,
		Technologies: technologies,
	}

	return project, nil
}

func (r *PostgresCVRepository) GetProjectByCV(resumeId uuid.UUID) ([]*domain.Project, error) {
	query := `
		SELECT id, name, description, repo_url, demo_url, start_date, end_date
		FROM projects
		WHERE resume_id = $1
		ORDER BY COALESCE(start_date, '9999-12-31') DESC
	`
	type projectRow struct {
		ID          uuid.UUID  `db:"id"`
		Name        string     `db:"name"`
		Description string     `db:"description"`
		RepoURL     string     `db:"repo_url"`
		DemoURL     string     `db:"demo_url"`
		StartDate   *time.Time `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
	}

	var rows []projectRow
	err := r.db.Select(&rows, query, resumeId)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("failed to get projects by resume")
		return nil, err
	}

	projects := make([]*domain.Project, len(rows))

	for i, row := range rows {
		var startDate, endDate string
		if row.StartDate != nil {
			startDate = row.StartDate.Format("2006-01-02")
		}
		if row.EndDate != nil {
			endDate = row.EndDate.Format("2006-01-02")
		} else {
			endDate = "Present"
		}

		technologies, err := r.GetProjectTechnologies(row.ID)

		if err != nil {
			log.Error().Err(err).Str("project_id", row.ID.String()).Msg("failed to get project technologies")
			continue
		}

		projects[i] = &domain.Project{Name: row.Name,
			Description:  row.Description,
			RepoURL:      row.RepoURL,
			DemoURL:      row.DemoURL,
			StartDate:    startDate,
			EndDate:      endDate,
			Technologies: technologies,
		}
	}
	return projects, nil
}

func (r *PostgresCVRepository) AddProjectTechnology(projectId uuid.UUID, technology string) error {
	query := `
		INSERT INTO project_technologies (id, project_id, technology)
		VALUES ($1, $2, $3)
	`

	id := uuid.New()
	_, err := r.db.Exec(query, id, projectId, technology)
	if err != nil {
		log.Error().Err(err).Msg("failed to add project tech")
	}
	return nil
}

func (r *PostgresCVRepository) GetProjectTechnologies(projectId uuid.UUID) ([]string, error) {
	query := `
		SELECT technology
		FROM project_technologies
		WHERE project_id = $1
		ORDER BY technology
	`

	var technologies []string
	err := r.db.Select(&technologies, query, projectId)
	if err != nil {
		log.Error().Err(err).Str("project_id", projectId.String()).Msg("failed to get project technologies")
		return nil, err
	}

	return technologies, nil
}

func (r *PostgresCVRepository) DeleteProjectTechnology(projectId uuid.UUID, technology string) error {
	query := `
		DELETE FROM project_technologies
		WHERE project_id = $1 AND technology = $2
	`

	result, err := r.db.Exec(query, projectId, technology)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete project technology")
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

func (r *PostgresCVRepository) AddCertification(resumeId uuid.UUID, certification *domain.Certification) (uuid.UUID, error) {
	query := `
		INSERT INTO certifications (
			id, resume_id, name, issuer, issue_date, 
			expiry_date, credential_id, url, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	// Apply BeforeSave to sanitize the data
	certification.BeforeSave()

	// Validate the entry
	if err := certification.Validate(); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	now := time.Now()

	// Parse dates
	var issueDate *time.Time
	var expiryDate *time.Time

	if certification.IssueDate != "" {
		parsedIssueDate, err := time.Parse("2006-01-02", certification.IssueDate)
		if err != nil {
			return uuid.Nil, err
		}
		issueDate = &parsedIssueDate
	}

	if certification.ExpiryDate != "" && certification.ExpiryDate != "No Expiration" {
		parsedExpiryDate, err := time.Parse("2006-01-02", certification.ExpiryDate)
		if err != nil {
			return uuid.Nil, err
		}
		expiryDate = &parsedExpiryDate
	}

	var returnedID uuid.UUID
	err := r.db.QueryRow(
		query,
		id,
		resumeId,
		certification.Name,
		certification.Issuer,
		issueDate,
		expiryDate,
		certification.CredentialID,
		certification.URL,
		now,
		now,
	).Scan(&returnedID)
	if err != nil {
		log.Error().Err(err).Msg("failed to add certification")
		return uuid.Nil, err
	}

	return returnedID, nil
}

func (r *PostgresCVRepository) UpdateCertification(id uuid.UUID, certification *domain.Certification) error {
	query := `
		UPDATE certifications
		SET name = $1,
			issuer = $2,
			issue_date = $3,
			expiry_date = $4,
			credential_id = $5,
			url = $6,
			updated_at = $7
		WHERE id = $8
	`

	// Apply BeforeSave to sanitize the data
	certification.BeforeSave()

	// Validate the entry
	if err := certification.Validate(); err != nil {
		return err
	}

	now := time.Now()

	// Parse dates
	var issueDate *time.Time
	var expiryDate *time.Time

	if certification.IssueDate != "" {
		parsedIssueDate, err := time.Parse("2006-01-02", certification.IssueDate)
		if err != nil {
			return err
		}
		issueDate = &parsedIssueDate
	}

	if certification.ExpiryDate != "" && certification.ExpiryDate != "No Expiration" {
		parsedExpiryDate, err := time.Parse("2006-01-02", certification.ExpiryDate)
		if err != nil {
			return err
		}
		expiryDate = &parsedExpiryDate
	}

	result, err := r.db.Exec(
		query,
		certification.Name,
		certification.Issuer,
		issueDate,
		expiryDate,
		certification.CredentialID,
		certification.URL,
		now,
		id,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to update certification")
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

func (r *PostgresCVRepository) DeleteCertification(id uuid.UUID) error {
	query := `
		DELETE FROM certifications
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete certification")
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

func (r *PostgresCVRepository) GetCertification(id uuid.UUID) (*domain.Certification, error) {
	query := `
		SELECT name, issuer, issue_date, expiry_date, credential_id, url
		FROM certifications
		WHERE id = $1
	`

	var certRow struct {
		Name         string     `db:"name"`
		Issuer       string     `db:"issuer"`
		IssueDate    time.Time  `db:"issue_date"`
		ExpiryDate   *time.Time `db:"expiry_date"`
		CredentialID string     `db:"credential_id"`
		URL          string     `db:"url"`
	}

	err := r.db.Get(&certRow, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("certification_id", id.String()).Msg("failed to get certification")
		return nil, err
	}

	// Format dates
	issueDate := certRow.IssueDate.Format("2006-01-02")
	var expiryDate string
	if certRow.ExpiryDate != nil {
		expiryDate = certRow.ExpiryDate.Format("2006-01-02")
	} else {
		expiryDate = "No Expiration"
	}

	certification := &domain.Certification{
		Name:         certRow.Name,
		Issuer:       certRow.Issuer,
		IssueDate:    issueDate,
		ExpiryDate:   expiryDate,
		CredentialID: certRow.CredentialID,
		URL:          certRow.URL,
	}

	return certification, nil
}

func (r *PostgresCVRepository) GetCertificationsByResume(resumeId uuid.UUID) ([]*domain.Certification, error) {
	query := `
		SELECT id, name, issuer, issue_date, expiry_date, credential_id, url
		FROM certifications
		WHERE resume_id = $1
		ORDER BY issue_date DESC
	`

	type certRow struct {
		ID           uuid.UUID  `db:"id"`
		Name         string     `db:"name"`
		Issuer       string     `db:"issuer"`
		IssueDate    time.Time  `db:"issue_date"`
		ExpiryDate   *time.Time `db:"expiry_date"`
		CredentialID string     `db:"credential_id"`
		URL          string     `db:"url"`
	}

	var rows []certRow
	err := r.db.Select(&rows, query, resumeId)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("Failed to get certifications by resume")
		return nil, err
	}

	certifications := make([]*domain.Certification, len(rows))
	for i, row := range rows {
		// Format dates
		issueDate := row.IssueDate.Format("2006-01-02")
		var expiryDate string
		if row.ExpiryDate != nil {
			expiryDate = row.ExpiryDate.Format("2006-01-02")
		} else {
			expiryDate = "No Expiration"
		}

		certifications[i] = &domain.Certification{
			Name:         row.Name,
			Issuer:       row.Issuer,
			IssueDate:    issueDate,
			ExpiryDate:   expiryDate,
			CredentialID: row.CredentialID,
			URL:          row.URL,
		}
	}

	return certifications, nil
}

// Experience
func (r *PostgresCVRepository) AddExperience(resumeId uuid.UUID, experience *domain.Experience) (uuid.UUID, error) {
	query := `
		INSERT INTO experience (
			id, resume_id, employer, job_title, location, 
			start_date, end_date, description, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	experience.BeforeSave()

	if err := experience.Validate(); err != nil {
		return uuid.Nil, err
	}

	id := uuid.New()
	now := time.Now()

	var startDate *time.Time
	var endDate *time.Time

	if experience.StartDate != "" {
		parsedStartDate, err := time.Parse("2006-01-02", experience.StartDate)
		if err != nil {
			return uuid.Nil, err
		}
		startDate = &parsedStartDate
	}

	if experience.EndDate != "" && experience.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", experience.EndDate)
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
		experience.Employer,
		experience.JobTitle,
		experience.Location,
		startDate,
		endDate,
		experience.Description,
		now,
		now).Scan(&returnedId)
	if err != nil {
		log.Error().Err(err).Msg("failed to add experience")
	}

	if len(experience.Achievements) > 0 {
		// TODO: Add achievements in a separate table if needed
	}

	return returnedId, nil
}

func (r *PostgresCVRepository) UpdateExperience(id uuid.UUID, experience *domain.Experience) error {
	query := `
		UPDATE experience
		SET employer = $1,
			job_title = $2,
			location = $3,
			start_date = $4,
			end_date = $5,
			description = $6,
			updated_at = $7
		WHERE id = $8
	`

	experience.BeforeSave()

	if err := experience.Validate(); err != nil {
		return err
	}

	now := time.Now()

	// Parse dates
	var startDate *time.Time
	var endDate *time.Time

	if experience.StartDate != "" {
		parsedStartDate, err := time.Parse("2006-01-02", experience.StartDate)
		if err != nil {
			return err
		}
		startDate = &parsedStartDate
	}

	if experience.EndDate != "" && experience.EndDate != "Present" {
		parsedEndDate, err := time.Parse("2006-01-02", experience.EndDate)
		if err != nil {
			return err
		}
		endDate = &parsedEndDate
	}

	result, err := r.db.Exec(
		query,
		experience.Employer,
		experience.JobTitle,
		experience.Location,
		startDate,
		endDate,
		experience.Description,
		now,
		id)

	if err != nil {
		log.Error().Err(err).Msg("failed to update experience")
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

func (r *PostgresCVRepository) DeleteExperience(id uuid.UUID) error {
	query := `
		DELETE FROM experience
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete experience")
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

func (r *PostgresCVRepository) GetExperience(id uuid.UUID) (*domain.Experience, error) {
	query := `
		SELECT employer, job_title, location, 
		       start_date, end_date, description
		FROM experience
		WHERE id = $1
	`

	var exp struct {
		Employer    string     `db:"employer"`
		JobTitle    string     `db:"job_title"`
		Location    string     `db:"location"`
		StartDate   time.Time  `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
		Description string     `db:"description"`
	}

	err := r.db.Get(&exp, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("experience_id", id.String()).Msg("failed to get experience")
		return nil, err
	}

	startDate := exp.StartDate.Format("2006-01-02")
	var endDate string
	if exp.EndDate != nil {
		endDate = exp.EndDate.Format("2006-01-02")
	} else {
		endDate = "Present"
	}

	experience := &domain.Experience{
		Employer:     exp.Employer,
		JobTitle:     exp.JobTitle,
		Location:     exp.Location,
		StartDate:    startDate,
		EndDate:      endDate,
		Description:  exp.Description,
		Achievements: []string{},
	}

	return experience, err
}

func (r *PostgresCVRepository) GetExperienceByResume(resumeId uuid.UUID) ([]*domain.Experience, error) {
	query := `
		SELECT id, employer, job_title, location, 
		       start_date, end_date, description
		FROM experience
		WHERE resume_id = $1
		ORDER BY start_date DESC
	`

	type experienceRow struct {
		ID          uuid.UUID  `db:"id"`
		Employer    string     `db:"employer"`
		JobTitle    string     `db:"job_title"`
		Location    string     `db:"location"`
		StartDate   time.Time  `db:"start_date"`
		EndDate     *time.Time `db:"end_date"`
		Description string     `db:"description"`
	}

	var rows []experienceRow
	err := r.db.Select(&rows, query, resumeId)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("Failed to get experience by resume")
		return nil, err
	}

	experience := make([]*domain.Experience, len(rows))

	for i, row := range rows {
		startDate := row.StartDate.Format("2006-01-02")
		var endDate string
		if row.EndDate != nil {
			endDate = row.EndDate.Format("2006-01-02")
		} else {
			endDate = "Present"
		}

		experience[i] = &domain.Experience{
			Employer:    row.Employer,
			JobTitle:    row.JobTitle,
			Location:    row.Location,
			StartDate:   startDate,
			EndDate:     endDate,
			Description: row.Description,
			// Fetch achievements if needed
			Achievements: []string{},
		}
	}

	return experience, nil
}

// Get Complete Resume
func (r *PostgresCVRepository) GetCompleteResume(resumeId uuid.UUID) (*domain.Resume, error) {
	resume, err := r.GetCVById(resumeId)
	if err != nil {
		return nil, err
	}

	personalInfo, err := r.GetPersonalInfo(resumeId)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if personalInfo != nil {
		resume.PersonalInfo = personalInfo
	}

	education, err := r.GetEducationByResume(resumeId)
	if err != nil {
		log.Error().Err(err).Str("resume_id", resumeId.String()).Msg("failed to get education")
	} else {
		resume.Education = education
	}

	return nil, nil
}
