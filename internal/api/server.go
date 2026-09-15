package api

import (
	"fmt"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Config     util.Config
	Store      *db.Store
	App        *fiber.App
	TokenMaker token.Maker
	SMS        *sms.Client
}

func NewServer(config util.Config, store *db.Store) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.JWT_SECRET)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})

	log.Info().
		Str("username", config.SMS_USERNAME).
		Bool("password_exists", config.SMS_PASSWORD != "").
		Bool("from_exists", config.SMS_FROM != "").
		Int("password_length", len(config.SMS_PASSWORD)).
		Int("from_length", len(config.SMS_FROM)).
		Msg("sunway config loaded")

	smsClient := sms.NewClient(
		config.SMS_USERNAME,
		config.SMS_PASSWORD,
		config.SMS_FROM,
	)

	return &Server{
		Config:     config,
		Store:      store,
		App:        app,
		TokenMaker: tokenMaker,
		SMS:        smsClient,
	}, nil
}

func (server *Server) Start(address string) error {
	return server.App.Listen(fmt.Sprintf(":%s", address))
}
