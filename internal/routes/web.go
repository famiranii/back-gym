package routes

import (
	"github.com/famiranii/back-gym.git/internal/api"
	"github.com/famiranii/back-gym.git/internal/api/handlers"
)

func SetupRoutes(server *api.Server) error {
	server.App.Get("/register", handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config).RegisterUser)
	server.App.Get("/login", handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config).LoginUser)
	return nil
}
	