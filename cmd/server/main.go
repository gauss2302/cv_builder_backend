package main

import (
	"context"
	"cv_builder/config"
	"cv_builder/internal/routes"
	"cv_builder/pkg/auth"
	database "cv_builder/pkg/db"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configs")
	}

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to db")
	}
	defer db.Close()

	redisOptions, err := redis.ParseURL(cfg.RedisUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get Redis URL")
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	jwtConfig := auth.JWTConfig{
		Secret:             cfg.JWTSecret,
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour, // 7 days
		ResetTokenExpiry:   1 * time.Hour,
		Issuer:             "resume_generator",
		Audience:           "resume_generator_users",
	}

	router := routes.SetupRoutes(db, redisClient, jwtConfig)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server..")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server was forced to shut down")
	}
	log.Info().Msg("server was done alright")
}
