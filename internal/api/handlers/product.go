package handlers

import (
	"strconv"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductHandler struct {
	Store      *db.Store
}

func NewProductHandler(store *db.Store) *ProductHandler {
	return &ProductHandler{
		Store:      store,
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

type UpdateImageRequest struct {
	ID        string `json:"id"`
	URL       string `json:"url" validate:"required"`
	IsPrimary bool   `json:"is_primary"`
}
type UpdateVariantRequests struct {
	ID    string `json:"id"` // اگه خالی بود → insert، اگه داشت → update
	Label string `json:"label" validate:"required"`
	Color string `json:"color"`
	Stock int32  `json:"stock"`
}
type UpdateProductRequest struct {
	Name        string                  `json:"name" validate:"required"`
	Description string                  `json:"description"`
	Price       float64                 `json:"price" validate:"required,gt=0"`
	Discount    float64                 `json:"discount"`
	CategoryID  string                  `json:"category_id"`
	IsActive    bool                    `json:"is_active"`
	Variants    []UpdateVariantRequests `json:"variants"`
	Images      []UpdateImageRequest    `json:"images"`
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
	if err := price.Scan(strconv.FormatFloat(req.Price, 'f', -1, 64)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid price"})
	}

	var discount pgtype.Numeric
	if err := discount.Scan(strconv.FormatFloat(req.Discount, 'f', -1, 64)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid discount"})
	}

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

	// variants
	existingVariants, err := h.Store.GetVariantsByProductID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	keepVariantIDs := map[uuid.UUID]bool{}
	for _, v := range req.Variants {
		if v.ID != "" {
			if vid, err := uuid.Parse(v.ID); err == nil {
				keepVariantIDs[vid] = true
			}
		}
	}

	for _, ev := range existingVariants {
		if !keepVariantIDs[ev.ID] {
			_ = h.Store.DeleteVariant(c.Context(), ev.ID)
		}
	}

	for _, v := range req.Variants {
		if v.ID == "" {
			_, err = h.Store.CreateVariant(c.Context(), db.CreateVariantParams{
				ProductID: id,
				Label:     v.Label,
				Color:     pgtype.Text{String: v.Color, Valid: v.Color != ""},
				Stock:     v.Stock,
			})
		} else {
			variantID, parseErr := uuid.Parse(v.ID)
			if parseErr != nil {
				continue
			}
			_, err = h.Store.UpdateVariant(c.Context(), db.UpdateVariantParams{
				ID:    variantID,
				Label: v.Label,
				Color: pgtype.Text{String: v.Color, Valid: v.Color != ""},
				Stock: v.Stock,
			})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// images
	existingImages, err := h.Store.GetProductImages(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	keepImageIDs := map[uuid.UUID]bool{}
	for _, img := range req.Images {
		if img.ID != "" {
			if imgID, err := uuid.Parse(img.ID); err == nil {
				keepImageIDs[imgID] = true
			}
		}
	}

	for _, ei := range existingImages {
		if !keepImageIDs[ei.ID] {
			_ = h.Store.DeleteProductImage(c.Context(), ei.ID)
		}
	}

	for _, img := range req.Images {
		if img.ID == "" {
			_, err = h.Store.CreateProductImage(c.Context(), db.CreateProductImageParams{
				ProductID: id,
				Url:       img.URL,
				IsPrimary: img.IsPrimary,
			})
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}
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
