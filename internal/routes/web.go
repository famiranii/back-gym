package routes

import (
	"github.com/famiranii/back-gym.git/internal/api"
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(server *api.Server) error {
	server.App.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Welcome to the Gym API!")
	})
	return nil
}
