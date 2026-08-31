package db

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) GetProductDetail(ctx context.Context, id uuid.UUID) (ProductDetailResponse, error) {
	product, err := s.GetProductByID(ctx, id)
	if err != nil {
		return ProductDetailResponse{}, err
	}

	categoryName := ""
	if product.CategoryID.Valid {
		catID := uuid.UUID(product.CategoryID.Bytes)
		cat, err := s.GetCategoryByID(ctx, catID)
		if err == nil {
			categoryName = cat.Name
		}
	}

	var (
		wg            sync.WaitGroup
		mu            sync.Mutex
		fetchErr      error
		images        []ProductImage
		variants      []ProductVariant
		averageRating float64
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		result, err := s.GetProductImages(ctx, id)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			fetchErr = err
			return
		}
		images = result
	}()

	go func() {
		defer wg.Done()
		result, err := s.GetVariantsByProductID(ctx, id)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			fetchErr = err
			return
		}
		variants = result
	}()

	go func() {
		defer wg.Done()
		result, err := s.GetAverageRating(ctx, id)
		mu.Lock()
		defer mu.Unlock()
		if err != nil || result == 0 {
			averageRating = 5
			return
		}
		averageRating = result
	}()

	wg.Wait()

	if fetchErr != nil {
		return ProductDetailResponse{}, fetchErr
	}

	return ProductDetailResponse{
		ID:            product.ID,
		Name:          product.Name,
		Description:   product.Description,
		Price:         product.Price,
		Discount:      product.Discount,
		FinalPrice:    product.FinalPrice,
		CategoryID:    product.CategoryID,
		CategoryName:  categoryName,
		IsActive:      product.IsActive,
		CreatedAt:     product.CreatedAt,
		UpdatedAt:     product.UpdatedAt,
		Images:        nullSlice(images),
		Variants:      nullSlice(variants),
		AverageRating: averageRating,
	}, nil
}

// جلوگیری از null در JSON — اگه slice خالیه، array خالی برگردون
func nullSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// pgtype.UUID به uuid.UUID
func pgUUIDToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}
