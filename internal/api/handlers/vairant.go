package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type VariantHandler struct {
	Store      *db.Store
	Config     util.Config
	TokenMaker token.Maker
}

func NewVariantHandler(store *db.Store, tokenMaker token.Maker, config util.Config) *VariantHandler {
	return &VariantHandler{
		Store:      store,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

type CreateVariantRequest struct {
	Label string `json:"label" validate:"required"`
	Stock int32  `json:"stock" validate:"gte=0"`
}

type UpdateVariantRequest struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
	Color string    `json:"color"`
	Stock int32     `json:"stock"`
}

func (h *VariantHandler) CreateVariant(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	var req CreateVariantRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	variant, err := h.Store.CreateVariant(c.Context(), db.CreateVariantParams{
		ProductID: productID,
		Label:     req.Label,
		Stock:     req.Stock,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(variant)
}

func (h *VariantHandler) GetVariantsByProduct(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	variants, err := h.Store.GetVariantsByProductID(c.Context(), productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(variants)
}

func (h *VariantHandler) UpdateVariantStock(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Stock int32 `json:"stock" validate:"gte=0"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	variant, err := h.Store.UpdateVariantStock(c.Context(), db.UpdateVariantStockParams{
		ID:    id,
		Stock: req.Stock,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(variant)
}

func (h *VariantHandler) DeleteVariant(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = h.Store.DeleteVariant(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
