package main

import (
	"context"
	"fmt"
	"os"

	"github.com/famiranii/back-gym.git/internal/api"
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/routes"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"

	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// ssl is for encrypting 
	
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		config.DB_USERNAME,
		config.DB_PASSWORD,
		config.DB_HOST,
		config.DB_PORT,
		config.DB_DATABASE)
	fmt.Println("DSN:", dsn)
	dbpool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
		os.Exit(1)
	}
	defer dbpool.Close()	

	runDBMigrations(config.MIGRATION_URL, dsn)

	store := db.NewStore(dbpool)

	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}
	if err := routes.SetupRoutes(server); err != nil {
		log.Fatal().Err(err).Msg("cannot create routes")
	}
	err = server.Start(config.APP_PORT)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}

func runDBMigrations(migraitionUrl string, dbSource string) {
	m, err := migrate.New(migraitionUrl, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}
	log.Info().Msg("db migrated successfully")
}
