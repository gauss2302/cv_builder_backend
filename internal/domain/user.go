package domain

import (
	"context"
	"github.com/google/uuid"
	tgInitData "github.com/telegram-mini-apps/init-data-golang"
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

type TelegramUser struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	User       tgInitData.User `json:"user" db:"user"`
	TelegramID int64           `json:"telegram_id" db:"telegram_id"`
	FirstName  string          `json:"first_name" db:"first_name"`
	LastName   string          `json:"last_name" db:"last_name"`
	Username   string          `json:"username" db:"username"`
	Role       string          `json:"role" db:"role"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

type TgUser struct {
	User tgInitData.User `json:"user"`
}

func (tg *TgUser) GetUserId() int64 {
	return tg.User.ID
}

type AuthResponse struct {
	Message      string          `json:"message"`
	QueryID      string          `json:"query_id,omitempty"`
	AuthDate     time.Time       `json:"auth_date,omitempty"`
	User         tgInitData.User `json:"user,omitempty"`
	Receiver     tgInitData.User `json:"receiver,omitempty"`
	Chat         tgInitData.Chat `json:"chat,omitempty"`
	StartParam   string          `json:"start_param,omitempty"`
	CanSendAfter *time.Time      `json:"can_send_after,omitempty"`
	ChatType     string          `json:"chat_type,omitempty"`
	ChatInstance string          `json:"chat_instance,omitempty"`
}

type TelegramAuthData struct {
	InitData      string
	ParsedData    *tgInitData.InitData
	IsValid       bool
	ValidationErr error
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
	CreateUser(ctx context.Context, user *User) error
	GetUserById(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*TelegramUser, error)
	CreateTelegramUser(ctx context.Context, userTg *TelegramUser) error

	CreateSession(ctx context.Context, session *Session) error
	GetSessionById(ctx context.Context, id uuid.UUID) (*Session, error)
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteUserSessions(ctx context.Context, userId uuid.UUID) error

	CreatePasswordReset(ctx context.Context, reset *PasswordReset) error
	GetPasswordResetByToken(ctx context.Context, token string) (*PasswordReset, error)
	MarkPasswordResetUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpiredPasswordReset(ctx context.Context) error
}
