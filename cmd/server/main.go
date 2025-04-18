package main

import (
	"cv_builder/config"
	database "cv_builder/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
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

}
