package config

import (
	"errors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DBUrl            string
	RedisUrl         string
	JWTSecret        string
	CSRFKey          string
	TelegramBotToken string
}

// Load loads configuration from environment variables with validation
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found, using environment variables")
	}

	config := &Config{
		Port:             os.Getenv("PORT"),
		DBUrl:            os.Getenv("DB_URL"),
		RedisUrl:         os.Getenv("REDIS_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		CSRFKey:          os.Getenv("CSRF_KEY"),
		TelegramBotToken: os.Getenv("MY_BOT_TOKEN"),
	}

	// Validate configuration
	var missingVars []string

	if config.Port == "" {
		// Default port if not specified
		config.Port = "8080"
	}

	if config.DBUrl == "" {
		missingVars = append(missingVars, "DB_URL")
	}

	if config.RedisUrl == "" {
		missingVars = append(missingVars, "REDIS_URL")
	}

	if config.JWTSecret == "" {
		missingVars = append(missingVars, "JWT_SECRET")
	}

	if config.CSRFKey == "" {
		missingVars = append(missingVars, "CSRF_KEY")
	}

	if len(missingVars) > 0 {
		return nil, errors.New("missing required environment variables: " + strings.Join(missingVars, ", "))
	}

	return config, nil
}
