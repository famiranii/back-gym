package util

import (
	"time"

	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
)

func AuthMiddleware(tokenMaker token.Maker) fiber.Handler {
	return func(c fiber.Ctx) error {
		accessToken := c.Cookies("access_token")

		// Access Token داریم
		if accessToken != "" {
			payload, err := tokenMaker.VerifyToken(accessToken)

			if err == nil {
				c.Locals("payload", payload)
				return c.Next()
			}
		}

		// Access Token نداریم یا منقضی شده
		refreshToken := c.Cookies("refresh_token")

		if refreshToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		// بررسی Refresh Token
		refreshPayload, err := tokenMaker.VerifyToken(refreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "refresh token is invalid or expired",
			})
		}

		// ساخت Access Token جدید
		newAccessToken, _, err := tokenMaker.CreateToken(
			refreshPayload.Phone,
			refreshPayload.UserID,
			refreshPayload.IsAdmin,
			15*time.Minute,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "could not create access token",
			})
		}

		// ذخیره Access Token جدید
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    newAccessToken,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
			Path:     "/",
		})

		c.Locals("payload", refreshPayload)

		return c.Next()
	}
}
