package handlers

import (
	"strconv"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ReviewHandler struct {
	Store *db.Store
}

func NewReviewHandler(store *db.Store) *ReviewHandler {
	return &ReviewHandler{
		Store: store,
	}
}

// =========================================================
// Request
// =========================================================

type UpsertReviewRequest struct {
	Rating *int16 `json:"rating"`
	Body   string `json:"body"`
}

// =========================================================
// POST /products/:id/reviews
// =========================================================

func (h *ReviewHandler) UpsertReview(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product_id",
		})
	}
	payload := c.Locals("payload").(*token.Payload)

	// User ID از JWT
	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	var req UpsertReviewRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// حداقل rating یا body باید وجود داشته باشد
	if req.Rating == nil && req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "rating or body required",
		})
	}

	// Validate rating
	if req.Rating != nil {
		if *req.Rating < 1 || *req.Rating > 5 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "rating must be between 1 and 5",
			})
		}
	}

	arg := db.UpsertReviewParams{
		ProductID: productID,
		UserID:    user.ID,
		Body: pgtype.Text{
			String: req.Body,
			Valid:  req.Body != "",
		},
	}

	if req.Rating != nil {
		arg.Rating = pgtype.Int2{
			Int16: *req.Rating,
			Valid: true,
		}
	}

	review, err := h.Store.UpsertReview(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save review",
		})
	}

	return c.Status(fiber.StatusOK).JSON(review)
}

// =========================================================
// GET /products/:id/reviews
// =========================================================

func (h *ReviewHandler) GetReviews(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product_id",
		})
	}

	reviews, err := h.Store.GetReviewsByProductID(
		c.Context(),
		productID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get reviews",
		})
	}

	return c.Status(fiber.StatusOK).JSON(reviews)
}

// =========================================================
// GET /products/:id/reviews/rating
// =========================================================

func (h *ReviewHandler) GetAverageRating(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product_id",
		})
	}

	averageRating, err := h.Store.GetAverageRating(
		c.Context(),
		productID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get average rating",
		})
	}

	return c.JSON(fiber.Map{
		"average_rating": averageRating,
	})
}

// =========================================================
// DELETE /products/:id/reviews
// =========================================================

func (h *ReviewHandler) DeleteReview(c fiber.Ctx) error {
	productID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid product_id",
		})
	}
	payload := c.Locals("payload").(*token.Payload)

	// User ID از JWT
	user, err := h.Store.GetUserByID(c.Context(), payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	err = h.Store.DeleteReview(
		c.Context(),
		db.DeleteReviewParams{
			ProductID: productID,
			UserID:    user.ID,
		},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete review",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =========================================================
// Admin - Get Pending Reviews
// =========================================================

// GET /admin/reviews/pending
func (h *ReviewHandler) GetPendingReviews(c fiber.Ctx) error {
	reviews, err := h.Store.GetPendingReviews(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get pending reviews",
		})
	}

	return c.Status(fiber.StatusOK).JSON(reviews)
}

// =========================================================
// Admin - Approve Review
// =========================================================

// PATCH /admin/reviews/:id/approve
func (h *ReviewHandler) ApproveReview(c fiber.Ctx) error {
	reviewID, err := parseReviewID(c)
	if err != nil {
		return err
	}

	review, err := h.Store.ApproveReview(
		c.Context(),
		reviewID,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "review not found or already processed",
		})
	}

	return c.Status(fiber.StatusOK).JSON(review)
}

// =========================================================
// Admin - Reject Review
// =========================================================

// PATCH /admin/reviews/:id/reject
func (h *ReviewHandler) RejectReview(c fiber.Ctx) error {
	reviewID, err := parseReviewID(c)
	if err != nil {
		return err
	}

	review, err := h.Store.RejectReview(
		c.Context(),
		reviewID,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "review not found or already processed",
		})
	}

	return c.Status(fiber.StatusOK).JSON(review)
}

// =========================================================
// Helpers
// =========================================================

func parseReviewID(c fiber.Ctx) (int64, error) {
	reviewID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid review_id"},
		)
		return 0, err
	}

	return reviewID, nil
}
