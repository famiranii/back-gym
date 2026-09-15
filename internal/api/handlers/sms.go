package handlers

import (
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/gofiber/fiber/v3"
)

// SMSHandler exposes endpoints for sending SMS through SunwaySMS.
type SMSHandler struct {
	Client *sms.Client
}

func NewSMSHandler(client *sms.Client) *SMSHandler {
	return &SMSHandler{Client: client}
}

type SendSMSRequest struct {
	To      []string `json:"to" validate:"required,min=1,max=1000,dive,required"`
	Message string   `json:"message" validate:"required"`
	Flash   bool     `json:"flash"`
}

// SendSMS delivers a message to one or more recipients.
func (h *SMSHandler) SendSMS(c fiber.Ctx) error {
	var req SendSMSRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	results, err := h.Client.Send(c.Context(), req.To, req.Message, req.Flash)
	if err != nil {
		if apiErr, ok := err.(*sms.APIError); ok {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": apiErr.Message,
				"code":  apiErr.Code,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
	})
}
