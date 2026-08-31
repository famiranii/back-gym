package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ReviewHandler struct {
	Store *db.Store
}

func NewReviewHandler(store *db.Store) *ReviewHandler {
	return &ReviewHandler{Store: store}
}

type UpsertReviewRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Rating *int16 `json:"rating"`
	Body   string `json:"body"`
}

// POST /products/:id/reviews
func (h *ReviewHandler) UpsertReview(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	var req UpsertReviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.Rating == nil && req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rating or body required"})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	arg := db.UpsertReviewParams{
		ProductID: productID,
		UserID:    userID,
		Body:      pgtype.Text{String: req.Body, Valid: req.Body != ""},
	}

	if req.Rating != nil {
		arg.Rating = pgtype.Int2{Int16: *req.Rating, Valid: true}
	}

	review, err := h.Store.UpsertReview(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(review)
}

// GET /products/:id/reviews
func (h *ReviewHandler) GetReviews(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	reviews, err := h.Store.GetReviewsByProductID(c.Context(), productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(reviews)
}

// DELETE /products/:id/reviews
func (h *ReviewHandler) DeleteReview(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product_id"})
	}

	userIDStr := c.Query("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	err = h.Store.DeleteReview(c.Context(), db.DeleteReviewParams{
		ProductID: productID,
		UserID:    userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
