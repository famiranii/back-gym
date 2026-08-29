package handlers

import (
	"strconv"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductHandler struct {
	Store      *db.Store
	Config     util.Config
	TokenMaker token.Maker
}

func NewProductHandler(store *db.Store, tokenMaker token.Maker, config util.Config) *ProductHandler {
	return &ProductHandler{
		Store:      store,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

type CreateProductRequest struct {
	Name        string                    `json:"name" validate:"required"`
	Description string                    `json:"description"`
	Price       float64                   `json:"price" validate:"required,gt=0"`
	Discount    float64                   `json:"discount"`
	CategoryID  string                    `json:"category_id"`
	IsActive    bool                      `json:"is_active"`
	Variants    []db.CreateProductVariant `json:"variants" validate:"required,min=1"`
	Images      []db.CreateProductImage   `json:"images"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Discount    float64 `json:"discount"`
	CategoryID  string  `json:"category_id"`
	IsActive    bool    `json:"is_active"`
}

func (h *ProductHandler) CreateProduct(c fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	arg := db.CreateProductWithVariantsParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Price:       req.Price,
		Discount:    req.Discount,
		IsActive:    req.IsActive,
		Variants:    req.Variants,
		Images:      req.Images,
	}

	if req.CategoryID != "" {
		catID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		arg.CategoryID = pgtype.UUID{Bytes: catID, Valid: true}
	}

	result, err := h.Store.CreateProductWithVariants(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *ProductHandler) UpdateProduct(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req UpdateProductRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var price pgtype.Numeric
	price.Scan(req.Price)

	var discount pgtype.Numeric
	discount.Scan(req.Discount)

	arg := db.UpdateProductParams{
		ID:          id,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Price:       price,
		Discount:    discount,
		IsActive:    req.IsActive,
	}

	if req.CategoryID != "" {
		catID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		arg.CategoryID = pgtype.UUID{Bytes: catID, Valid: true}
	}

	product, err := h.Store.UpdateProduct(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(product)
}

// GetProduct — محصول کامل با تصاویر و variants
func (h *ProductHandler) GetProduct(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	detail, err := h.Store.GetProductDetail(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	return c.JSON(detail)
}

func (h *ProductHandler) GetAllProducts(c fiber.Ctx) error {
	limit := int32(10)
	offset := int32(0)

	if value := c.Query("limit"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			limit = int32(parsed)
		}
	}

	if value := c.Query("offset"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			offset = int32(parsed)
		}
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	products, err := h.Store.GetAllProducts(c.Context(), db.GetAllProductsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(products)
}

func (h *ProductHandler) DeleteProduct(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.Store.DeleteProduct(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
