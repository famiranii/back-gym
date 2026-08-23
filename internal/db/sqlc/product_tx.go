package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateProductVariant struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Stock int32  `json:"stock"`
}

type CreateProductImage struct {
	URL       string `json:"url"`
	IsPrimary bool   `json:"is_primary"`
}

type CreateProductWithVariantsParams struct {
	Name        string
	Description pgtype.Text
	Price       float64
	Discount    float64
	CategoryID  pgtype.UUID
	IsActive    bool
	Variants    []CreateProductVariant
	Images      []CreateProductImage
}

type CreateProductWithVariantsResult struct {
	Product  Product          `json:"product"`
	Variants []ProductVariant `json:"variants"`
	Images   []ProductImage   `json:"images"`
}

func (s *Store) CreateProductWithVariants(ctx context.Context, arg CreateProductWithVariantsParams) (CreateProductWithVariantsResult, error) {
	var result CreateProductWithVariantsResult

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)

	q := New(tx)

	var price pgtype.Numeric
	if err := price.Scan(fmt.Sprintf("%.2f", arg.Price)); err != nil {
		return result, err
	}

	var discount pgtype.Numeric
	if err := discount.Scan(fmt.Sprintf("%.2f", arg.Discount)); err != nil {
		return result, err
	}

	product, err := q.CreateProduct(ctx, CreateProductParams{
		Name:        arg.Name,
		Description: arg.Description,
		Price:       price,
		Discount:    discount,
		CategoryID:  arg.CategoryID,
		IsActive:    arg.IsActive,
	})
	if err != nil {
		return result, err
	}
	result.Product = product

	for _, v := range arg.Variants {
		variant, err := q.CreateVariant(ctx, CreateVariantParams{
			ProductID: product.ID,
			Label:     v.Label,
			Color:     pgtype.Text{String: v.Color, Valid: v.Color != ""},
			Stock:     v.Stock,
		})
		if err != nil {
			return result, err
		}
		result.Variants = append(result.Variants, variant)
	}

	for _, img := range arg.Images {
		image, err := q.CreateProductImage(ctx, CreateProductImageParams{
			ProductID: product.ID,
			Url:       img.URL,
			IsPrimary: img.IsPrimary,
		})
		if err != nil {
			return result, err
		}
		result.Images = append(result.Images, image)
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}

	return result, nil
}
