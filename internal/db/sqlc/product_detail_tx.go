package db

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) GetProductDetail(ctx context.Context, id uuid.UUID) (ProductDetailResponse, error) {
	// ── ۱. محصول پایه ─────────────────────────────────────────────
	product, err := s.GetProductByID(ctx, id)
	if err != nil {
		return ProductDetailResponse{}, err
	}

	// ── ۲. category ───────────────────────────────────────────────
	categoryName := ""
	if product.CategoryID.Valid {
		catID := uuid.UUID(product.CategoryID.Bytes)
		cat, err := s.GetCategoryByID(ctx, catID)
		if err == nil {
			categoryName = cat.Name
		}
	}

	// ── ۳. images و variants موازی ────────────────────────────────
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		fetchErr error
		images   []ProductImage
		variants []ProductVariant
	)

	wg.Add(2)

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

	wg.Wait()

	if fetchErr != nil {
		return ProductDetailResponse{}, fetchErr
	}

	// ── ۴. خروجی ─────────────────────────────────────────────────
	return ProductDetailResponse{
		ID:           product.ID,
		Name:         product.Name,
		Description:  product.Description,
		Price:        product.Price,
		Discount:     product.Discount,
		CategoryID:   product.CategoryID,
		CategoryName: categoryName,
		IsActive:     product.IsActive,
		CreatedAt:    product.CreatedAt,
		UpdatedAt:    product.UpdatedAt,
		Images:       nullSlice(images),
		Variants:     nullSlice(variants),
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
