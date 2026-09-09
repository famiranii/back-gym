package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type WishlistHandler struct {
	Store *db.Store
}

func NewWishlistHandler(store *db.Store) *WishlistHandler {
	return &WishlistHandler{Store: store}
}

func getUserID(c fiber.Ctx) (uuid.UUID, error) {
	payload := c.Locals("payload").(*token.Payload)
	return payload.UserID, nil
}

// GET /wishlist
func (h *WishlistHandler) GetWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	items, err := h.Store.GetWishlistByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if items == nil {
		items = []db.GetWishlistByUserIDRow{}
	}

	return c.JSON(items)
}

// POST /wishlist
func (h *WishlistHandler) ToggleWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		ProductID string `json:"product_id"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	productID, err := uuid.Parse(body.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	exists, err := h.Store.IsInWishlist(c.Context(), db.IsInWishlistParams{
		UserID:    userID,
		ProductID: productID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if exists {
		err = h.Store.RemoveFromWishlist(c.Context(), db.RemoveFromWishlistParams{
			UserID:    userID,
			ProductID: productID,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"action": "removed" , "id" : productID})
	}

	_, err = h.Store.AddToWishlist(c.Context(), db.AddToWishlistParams{
		UserID:    userID,
		ProductID: productID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"action": "added" , "id" : productID})
}

// DELETE /wishlist/:product_id
func (h *WishlistHandler) RemoveFromWishlist(c fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	productID, err := uuid.Parse(c.Params("product_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	err = h.Store.RemoveFromWishlist(c.Context(), db.RemoveFromWishlistParams{
		UserID:    userID,
		ProductID: productID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
