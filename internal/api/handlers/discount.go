package handlers

import (
	"strings"
	"time"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type DiscountCodeHandler struct {
	store *db.Store
}

func NewDiscountCodeHandler(store *db.Store) *DiscountCodeHandler {
	return &DiscountCodeHandler{
		store: store,
	}
}

type CreateDiscountCodeRequest struct {
	Code              string `json:"code"`
	DiscountType      string `json:"discount_type"`
	DiscountValue     int64  `json:"discount_value"`
	MinOrderAmount    int64  `json:"min_order_amount"`
	MaxDiscountAmount *int64 `json:"max_discount_amount"`
	UsageLimit        *int32 `json:"usage_limit"`
	StartsAt          string `json:"starts_at"`
	ExpiresAt         string `json:"expires_at"`
}

func (h *DiscountCodeHandler) Create(c fiber.Ctx) error {
	var req CreateDiscountCodeRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	code := strings.TrimSpace(strings.ToUpper(req.Code))

	if code == "" {
		generatedCode, err := util.GenerateDiscountCode(8)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "خطا در تولید کد تخفیف",
			})
		}

		code = generatedCode
	}

	// اینجا باید timestamp ها را تبدیل کنیم
	var startsAt pgtype.Timestamptz
	var expiresAt pgtype.Timestamptz

	if req.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartsAt)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "تاریخ شروع نامعتبر است",
			})
		}

		startsAt = pgtype.Timestamptz{
			Time:  t,
			Valid: true,
		}
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "تاریخ پایان نامعتبر است",
			})
		}

		expiresAt = pgtype.Timestamptz{
			Time:  t,
			Valid: true,
		}
	}

	discount, err := h.store.CreateDiscountCode(c.Context(), db.CreateDiscountCodeParams{
		Code:           code,
		DiscountType:   req.DiscountType,
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscountAmount: pgtype.Int8{
			Int64: func() int64 {
				if req.MaxDiscountAmount == nil {
					return 0
				}
				return *req.MaxDiscountAmount
			}(),
			Valid: req.MaxDiscountAmount != nil,
		},
		UsageLimit: pgtype.Int4{
			Int32: func() int32 {
				if req.UsageLimit == nil {
					return 0
				}
				return *req.UsageLimit
			}(),
			Valid: req.UsageLimit != nil,
		},
		StartsAt:  startsAt,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"code": discount.Code,
	})
}

type ValidateDiscountRequest struct {
	Code     string `json:"code"`
	Subtotal int64  `json:"subtotal"`
}

func (h *DiscountCodeHandler) ValidateDiscount(c fiber.Ctx) error {
	var req ValidateDiscountRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"valid": false,
			"error": "invalid request",
		})
	}

	discount, err := h.store.GetValidDiscountCode(
		c.Context(),
		req.Code,
	)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"valid": false,
			"error": "invalid discount code",
		})
	}

	if req.Subtotal < discount.MinOrderAmount {
		return c.Status(400).JSON(fiber.Map{
			"valid": false,
			"error": "minimum order amount not met",
		})
	}

	discountAmount := calculateDiscount(discount, req.Subtotal)

	return c.JSON(fiber.Map{
		"valid":           true,
		"code":            discount.Code,
		"discount_type":   discount.DiscountType,
		"discount_value":  discount.DiscountValue,
		"discount_amount": discountAmount,
		"final_subtotal":  req.Subtotal - discountAmount,
	})
}

func (h *DiscountCodeHandler) List(c fiber.Ctx) error {
	items, err := h.store.ListDiscountCodes(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(items)
}

func (h *DiscountCodeHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "شناسه کد تخفیف نامعتبر است",
		})
	}

	err = h.store.DeleteDiscountCode(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "خطا در حذف کد تخفیف",
		})
	}

	return c.JSON(fiber.Map{
		"message": "کد تخفیف با موفقیت حذف شد",
	})
}
