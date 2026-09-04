package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CategoryHandler struct {
	Store      *db.Store
	Config     util.Config
	TokenMaker token.Maker
}

func NewCategoryHandler(store *db.Store, tokenMaker token.Maker, config util.Config) *CategoryHandler {
	return &CategoryHandler{
		Store:      store,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

type CreateCategoryRequest struct {
	Name     string `json:"name" validate:"required"`
	ParentID string `json:"parent_id"`
	ImageUrl string `json:"image_url"`
}

func (h *CategoryHandler) GetAllCategories(c fiber.Ctx) error {
	categories, err := h.Store.GetAllCategories(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(categories)
}

func (h *CategoryHandler) GetCategoryByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	category, err := h.Store.GetCategoryByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "category not found"})
	}
	return c.JSON(category)
}

func (h *CategoryHandler) CreateCategory(c fiber.Ctx) error {
	var req CreateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	arg := db.CreateCategoryParams{
		Name:     req.Name,
		ImageUrl: pgtype.Text{String: req.ImageUrl, Valid: req.ImageUrl != ""},
	}

	if req.ParentID != "" {
		parentID, err := uuid.Parse(req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid parent_id"})
		}
		arg.ParentID = pgtype.UUID{Bytes: parentID, Valid: true}
	}

	category, err := h.Store.CreateCategory(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(category)
}

func (h *CategoryHandler) DeleteCategory(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = h.Store.DeleteCategory(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
