package api

import (
	"fmt"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
)

type Server struct {
	Config     util.Config
	Store      *db.Store
	App        *fiber.App
	TokenMaker token.Maker
}

func NewServer(config util.Config, store *db.Store) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.JWT_SECRET)
    if err != nil {
        return nil, fmt.Errorf("cannot create token maker: %w", err)
    }
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})
	return &Server{
		Config:     config,
		Store:      store,
		App:        app,
		TokenMaker: tokenMaker,
	}, nil
}

func (server *Server) Start(address string) error {
	return server.App.Listen(fmt.Sprintf(":%s", address))
}
