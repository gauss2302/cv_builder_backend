package routes

import (
	"cv_builder/internal/handler"
	"cv_builder/internal/repository"
	"cv_builder/internal/service"
	"cv_builder/pkg/auth"
	"cv_builder/pkg/security"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

func SetupRoutes(db *sqlx.DB, redisClient *redis.Client, jwtConfig auth.JWTConfig) http.Handler {
	corsMiddleware := security.CORSMiddleware(security.DefaultCORSConfig())
	mux := http.NewServeMux()

	userRepo := repository.NewPostgresUserRepository(db)

	jwtHandler := auth.NewJWT(jwtConfig)

	authServiceConfig := service.AuthServiceConfig{
		AccessTokenExpiry:  jwtConfig.AccessTokenExpiry,
		RefreshTokenExpiry: jwtConfig.RefreshTokenExpiry,
		ResetTokenExpiry:   jwtConfig.ResetTokenExpiry,
	}

	authService := service.NewAuthService(userRepo, jwtHandler, authServiceConfig)

	//authMiddleware := handler.NewAuthMiddleware(authService)

	authHandler := handler.NewAuthHandler(authService, redisClient)

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		handler.RespondWithJSON(w, http.StatusOK, map[string]any{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /api/v1/register", authHandler.RegisterHandler)
	mux.HandleFunc("POST /api/v1/login", authHandler.LoginHandler)
	mux.HandleFunc("POST /api/v1/refresh-token", authHandler.RefreshTokenHandler)
	mux.HandleFunc("POST /api/v1/logout", authHandler.LogoutHandler)
	mux.HandleFunc("POST /api/v1/request-password-reset", authHandler.RequestPasswordResetHandler)
	mux.HandleFunc("POST /api/v1/reset-password", authHandler.ResetPasswordHandler)

	handlerWithCORS := corsMiddleware(mux)

	return handlerWithCORS
}
