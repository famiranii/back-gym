// internal/handlers/cart_handler.go

package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type CartHandler struct {
	Store *db.Store
}

func NewCartHandler(store *db.Store) *CartHandler {
	return &CartHandler{Store: store}
}

// GET /cart
func (h *CartHandler) GetCart(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	items, err := h.Store.GetCart(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(items)
}

// POST /cart
type AddToCartRequest struct {
	VariantID string `json:"variant_id" validate:"required"`
	Quantity  int32  `json:"quantity" validate:"required,gt=0"`
}

func (h *CartHandler) AddToCart(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	var req AddToCartRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid variant_id"})
	}

	item, err := h.Store.AddToCart(c.Context(), db.AddToCartParams{
		UserID:    user.ID,
		VariantID: variantID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(item)
}

// PATCH /cart/:id
type UpdateCartRequest struct {
	Quantity int32 `json:"quantity" validate:"required,gt=0"`
}

func (h *CartHandler) UpdateCartItem(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req UpdateCartRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	item, err := h.Store.UpdateCartItemQuantity(c.Context(), db.UpdateCartItemQuantityParams{
		ID:       itemID,
		UserID:   user.ID,
		Quantity: req.Quantity,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(item)
}

// DELETE /cart/:id
func (h *CartHandler) RemoveFromCart(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = h.Store.RemoveFromCart(c.Context(), db.RemoveFromCartParams{
		ID:     itemID,
		UserID: user.ID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DELETE /cart
func (h *CartHandler) ClearCart(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	err = h.Store.ClearCart(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
