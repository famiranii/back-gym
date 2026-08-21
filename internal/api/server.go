package api

import (
	"fmt"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
)

type Server struct {
	config util.Config
	Store  *db.Store
	App    *fiber.App
}

func NewServer(config util.Config, store *db.Store) (*Server, error) {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})
	return &Server{
		config: config,
		Store:  store,
		App:    app,
	}, nil
}

func (server *Server) Start(address string) error {
	return server.App.Listen(fmt.Sprintf(":%s", address))
}
