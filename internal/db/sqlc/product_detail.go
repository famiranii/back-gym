package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductDetailResponse — خروجی کامل یک محصول برای API
type ProductDetailResponse struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description pgtype.Text      `json:"description"`
	Price       pgtype.Numeric   `json:"price"`
	Discount    pgtype.Numeric   `json:"discount"`
	CategoryID  pgtype.UUID      `json:"category_id"`
	CategoryName string          `json:"category_name"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   pgtype.Timestamp `json:"created_at"`
	UpdatedAt   pgtype.Timestamp `json:"updated_at"`
	Images      []ProductImage   `json:"images"`
	Variants    []ProductVariant `json:"variants"`
}
