package routes

import (
	"cv_builder/pkg/auth"
	"cv_builder/pkg/security"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"net/http"
)

func SetupRoutes(db *sqlx.DB, redisClient *redis.Client, jwtConfig auth.JWTConfig) http.Handler {
	corsMiddleware := security.CORSMiddleware()
}
