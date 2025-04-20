package domain

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Session struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	ClientIP     string    `json:"client_ip" db:"client_ip"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type PasswordReset struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UsedAt    time.Time `json:"used_at,omitempty" db:"used_at"`
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserById(id uuid.UUID) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id uuid.UUID) error

	CreateSession(session *Session) error
	GetSessionById(id uuid.UUID) (*Session, error)
	GetSessionByToken(token string) (*Session, error)
	DeleteSession(id uuid.UUID) error
	DeleteUserSessions(userId uuid.UUID) error

	CreatePasswordReset(reset *PasswordReset) error
	GetPasswordResetByToken(token string) (*PasswordReset, error)
	MarkPasswordResetUsed(id uuid.UUID) error
	DeleteExpiredPasswordReset() error
}
