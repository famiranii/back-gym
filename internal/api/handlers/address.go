// internal/handlers/address_handler.go

package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AddressHandler struct {
	Store *db.Store
}

func NewAddressHandler(store *db.Store) *AddressHandler {
	return &AddressHandler{Store: store}
}

type AddressRequest struct {
	Title      string  `json:"title"`
	Province   string  `json:"province"`
	City       string  `json:"city"`
	Address    string  `json:"address"`
	PostalCode string  `json:"postal_code"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	IsDefault  bool    `json:"is_default"`
}

// GET /users/:id/addresses
func (h *AddressHandler) GetAddresses(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	addresses, err := h.Store.GetUserAddresses(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(addresses)
}

// POST /users/:id/addresses
func (h *AddressHandler) CreateAddress(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	var req AddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	arg := db.CreateUserAddressParams{
		UserID:     userID,
		Title:      pgtype.Text{String: req.Title, Valid: req.Title != ""},
		Province:   pgtype.Text{String: req.Province, Valid: req.Province != ""},
		City:       pgtype.Text{String: req.City, Valid: req.City != ""},
		Address:    pgtype.Text{String: req.Address, Valid: req.Address != ""},
		PostalCode: pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Lat:        pgtype.Numeric{},
		Lng:        pgtype.Numeric{},
		IsDefault:  req.IsDefault,
	}

	if req.Lat != 0 {
		arg.Lat.Scan(req.Lat)
	}
	if req.Lng != 0 {
		arg.Lng.Scan(req.Lng)
	}

	address, err := h.Store.CreateUserAddress(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(address)
}

// PUT /users/:id/addresses/:addr_id
func (h *AddressHandler) UpdateAddress(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	addrID, err := uuid.Parse(c.Params("addr_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid address_id"})
	}

	var req AddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	arg := db.UpdateUserAddressParams{
		ID:         addrID,
		UserID:     userID,
		Title:      pgtype.Text{String: req.Title, Valid: req.Title != ""},
		Province:   pgtype.Text{String: req.Province, Valid: req.Province != ""},
		City:       pgtype.Text{String: req.City, Valid: req.City != ""},
		Address:    pgtype.Text{String: req.Address, Valid: req.Address != ""},
		PostalCode: pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Lat:        pgtype.Numeric{},
		Lng:        pgtype.Numeric{},
		IsDefault:  req.IsDefault,
	}

	if req.Lat != 0 {
		arg.Lat.Scan(req.Lat)
	}
	if req.Lng != 0 {
		arg.Lng.Scan(req.Lng)
	}

	address, err := h.Store.UpdateUserAddress(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(address)
}

// DELETE /users/:id/addresses/:addr_id
func (h *AddressHandler) DeleteAddress(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	addrID, err := uuid.Parse(c.Params("addr_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid address_id"})
	}

	err = h.Store.DeleteUserAddress(c.Context(), db.DeleteUserAddressParams{
		ID:     addrID,
		UserID: userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// PATCH /users/:id/addresses/:addr_id/default
func (h *AddressHandler) SetDefault(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	addrID, err := uuid.Parse(c.Params("addr_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid address_id"})
	}

	err = h.Store.SetDefaultAddress(c.Context(), db.SetDefaultAddressParams{
		ID:     addrID,
		UserID: userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}