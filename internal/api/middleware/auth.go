package middleware

import (
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
)

func AuthMiddleware(tokenMaker token.Maker) fiber.Handler {
	return func(c fiber.Ctx) error {
		accessToken := c.Cookies("access_token")

		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "access token is required",
			})
		}

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Locals("payload", payload)

		return c.Next()
	}
}

func OptionalAuthMiddleware(tokenMaker token.Maker) fiber.Handler {
	return func(c fiber.Ctx) error {
		accessToken := c.Cookies("access_token")

		// Token وجود ندارد؛ مهمان هستیم
		if accessToken == "" {
			return c.Next()
		}

		// Token وجود دارد ولی معتبر نیست
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Locals("payload", payload)

		return c.Next()
	}
}

func AdminMiddleware(tokenMaker token.Maker) fiber.Handler {
	return func(c fiber.Ctx) error {
		accessToken := c.Cookies("access_token")
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "access token is required"})
		}

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		if !payload.IsAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied"})
		}

		c.Locals("payload", payload)
		return c.Next()
	}
}
