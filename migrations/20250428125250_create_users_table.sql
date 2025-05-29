-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table stores authentication and basic user information
-- Supports both email/password and Telegram authentication
CREATE TABLE users (
                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Email authentication fields
                       email TEXT UNIQUE,
                       password_hash TEXT,

    -- Telegram authentication fields
                       telegram_id BIGINT UNIQUE,
                       first_name TEXT,
                       last_name TEXT,
                       username TEXT,
                       photo_url TEXT,
                       language_code TEXT,
                       is_premium BOOLEAN DEFAULT FALSE,
                       allows_write_to_pm BOOLEAN DEFAULT FALSE,

    -- Common fields
                       role TEXT NOT NULL DEFAULT 'user',
                       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure user has either email or telegram_id (but not necessarily both)
                       CONSTRAINT check_user_auth_method
                           CHECK (email IS NOT NULL OR telegram_id IS NOT NULL),

    -- If email exists, password_hash must exist (for email users)
                       CONSTRAINT check_email_password
                           CHECK (email IS NULL OR password_hash IS NOT NULL)
);

-- Indexes for faster lookups
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_telegram_id ON users(telegram_id) WHERE telegram_id IS NOT NULL;
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Partial unique index to allow multiple NULL values but ensure uniqueness for non-NULL values
CREATE UNIQUE INDEX idx_users_email_unique ON users(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX idx_users_telegram_id_unique ON users(telegram_id) WHERE telegram_id IS NOT NULL;

-- Add comprehensive comments
COMMENT ON TABLE users IS 'Stores user authentication and basic information for both email and Telegram users';
COMMENT ON COLUMN users.id IS 'Unique identifier for the user';
COMMENT ON COLUMN users.email IS 'User email address for email authentication (NULL for Telegram-only users)';
COMMENT ON COLUMN users.password_hash IS 'Hashed password for email authentication (NULL for Telegram-only users)';
COMMENT ON COLUMN users.telegram_id IS 'Telegram user ID for Telegram authentication (NULL for email-only users)';
COMMENT ON COLUMN users.first_name IS 'User first name (from Telegram or manually entered)';
COMMENT ON COLUMN users.last_name IS 'User last name (from Telegram or manually entered)';
COMMENT ON COLUMN users.username IS 'Telegram username (without @)';
COMMENT ON COLUMN users.photo_url IS 'URL to user profile photo from Telegram';
COMMENT ON COLUMN users.language_code IS 'User language code from Telegram';
COMMENT ON COLUMN users.is_premium IS 'Whether user has Telegram Premium';
COMMENT ON COLUMN users.allows_write_to_pm IS 'Whether user allows writing to PM on Telegram';
COMMENT ON COLUMN users.role IS 'User role (user, admin, etc.)';
COMMENT ON COLUMN users.created_at IS 'Timestamp when the user account was created';
COMMENT ON COLUMN users.updated_at IS 'Timestamp when the user account was last updated';

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_telegram_id_unique;
DROP INDEX IF EXISTS idx_users_email_unique;
DROP INDEX IF EXISTS idx_users_telegram_id;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;