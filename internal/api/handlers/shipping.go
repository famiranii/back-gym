package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/gofiber/fiber/v3"
)

type ShippingHandler struct {
	store *db.Store
}

func NewShippingHandler(store *db.Store) *ShippingHandler {
	return &ShippingHandler{store: store}
}

func (h *ShippingHandler) GetShippingCost(c fiber.Ctx) error {
	cost, err := h.store.GetShippingCost(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get shipping cost"})
	}
	return c.JSON(fiber.Map{"cost": cost})
}

func (h *ShippingHandler) UpdateShippingCost(c fiber.Ctx) error {
	var req struct {
		Cost int64 `json:"cost"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	setting, err := h.store.UpdateShippingCost(c.Context(), req.Cost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update shipping cost"})
	}
	return c.JSON(setting)
}