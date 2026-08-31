package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductDetailResponse struct {
	ID            uuid.UUID        `json:"id"`
	Name          string           `json:"name"`
	Description   pgtype.Text      `json:"description"`
	Price         pgtype.Numeric   `json:"price"`
	Discount      pgtype.Numeric   `json:"discount"`
	FinalPrice    int32            `json:"final_price"`
	CategoryID    pgtype.UUID      `json:"category_id"`
	CategoryName  string           `json:"category_name"`
	IsActive      bool             `json:"is_active"`
	CreatedAt     pgtype.Timestamp `json:"created_at"`
	UpdatedAt     pgtype.Timestamp `json:"updated_at"`
	Images        []ProductImage   `json:"images"`
	Variants      []ProductVariant `json:"variants"`
	AverageRating float64          `json:"average_rating"`
}