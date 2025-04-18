package main

import (
	"context"
	"cv_builder/config"
	database "cv_builder/pkg/db"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
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
}
