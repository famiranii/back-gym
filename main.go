package main

import (
	"context"
	"fmt"
	"os"

	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.APP_DEBUG == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		config.DB_USERNAME,
		config.DB_PASSWORD,
		config.DB_HOST,
		config.DB_PORT,
		config.DB_DATABASE)
	dbpool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
		os.Exit(1)
	}
	defer dbpool.Close()
}
