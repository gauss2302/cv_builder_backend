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

var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record already exists")
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresUserRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	//default values
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.Role == "" {
		user.Role = "user" // Default role
	}
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	var id uuid.UUID
	err := r.db.QueryRow(query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt).Scan(&id)
	if err != nil {
		if isDubpicateKeyError(err) {
			log.Error().Err(err).Str("email", user.Email).Msg("cannot create user with the same email")
			return ErrConflict
		}
		log.Error().Err(err).Msg("failed to create a user")
		return err
	}
	return nil
}

func (r *PostgresRepository) GetUserById(id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := r.db.Get(&user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("user_id", id.String()).Msg("failed to get user by id")
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) GetUserByEmail(email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := r.db.Get(&user, query, email)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error().Err(err).Str("email", email).Msg("failed to get user by email")
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUser(user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, role = $3, updated_at = $4
		WHERE id = $5
	`

	user.UpdatedAt = time.Now()

	result, err := r.db.Exec(
		query,
		user.Email, user.PasswordHash, user.Role, user.UpdatedAt, user.ID)
	if err != nil {
		if isDubpicateKeyError(err) {
			log.Error().Err(err).Str("email", user.Email).Msg("failed to update user: same email found")
			return ErrConflict
		}
		log.Error().Err(err).Msg("failed to update user")
		return err
	}
	rowsAffected, err := result.RowsAffected()

	if err != nil {
		log.Error().Err(err).Msg("failed to change rows")
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Error().Err(err).Str("user_id", id.String()).Msg("failed to delete user")
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

func (r *PostgresRepository) MarkPasswordResetUsed(id uuid.UUID) error {
	query := `UPDATE password_resets SET user_at = $1 WHERE id = $2`

	now := time.Now()
	result, err := r.db.Exec(query, now, id)
	if err != nil {
		log.Error().Err(err).Str("reset_id", id.String()).Msg("failed to mark pwd reset")
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

func (r *PostgresRepository) DeleteExpiredPasswordResets() error {
	query := `
		DELETE FROM password_resets
		WHERE expires_at < $1
		OR used_at IS NOT NULL
	`

	now := time.Now()
	_, err := r.db.Exec(query, now)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete expired password resets")
		return err
	}
	return nil
}

func isDubpicateKeyError(err error) bool {
	return err != nil && err.Error() != "" &&
		(err.Error() == "pq: duplicate key value violates unique constraint" ||
			err.Error() == "ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)")
}
