package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type BannerHandler struct {
	Store      *db.Store
	Config     util.Config
	TokenMaker token.Maker
}

func NewBannerHandler(store *db.Store, tokenMaker token.Maker, config util.Config) *BannerHandler {
	return &BannerHandler{
		Store:      store,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

type CreateBannerRequest struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	ButtonText string `json:"button_text"`
	ButtonURL  string `json:"button_url"`
	ImageURL   string `json:"image_url" validate:"required"`
	IsActive   bool   `json:"is_active"`
	SortOrder  int32  `json:"sort_order"`
}

func (h *BannerHandler) GetActiveBanners(c fiber.Ctx) error {
	banners, err := h.Store.GetActiveBanners(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(banners)
}

func (h *BannerHandler) CreateBanner(c fiber.Ctx) error {
	var req CreateBannerRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	banner, err := h.Store.CreateBanner(c.Context(), db.CreateBannerParams{
		Title:      pgtype.Text{String: req.Title, Valid: req.Title != ""},
		Subtitle:   pgtype.Text{String: req.Subtitle, Valid: req.Subtitle != ""},
		ButtonText: pgtype.Text{String: req.ButtonText, Valid: req.ButtonText != ""},
		ButtonUrl:  pgtype.Text{String: req.ButtonURL, Valid: req.ButtonURL != ""},
		ImageUrl:   req.ImageURL,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(banner)
}

func (h *BannerHandler) UpdateBanner(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req CreateBannerRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	banner, err := h.Store.UpdateBanner(c.Context(), db.UpdateBannerParams{
		ID:         id,
		Title:      pgtype.Text{String: req.Title, Valid: req.Title != ""},
		Subtitle:   pgtype.Text{String: req.Subtitle, Valid: req.Subtitle != ""},
		ButtonText: pgtype.Text{String: req.ButtonText, Valid: req.ButtonText != ""},
		ButtonUrl:  pgtype.Text{String: req.ButtonURL, Valid: req.ButtonURL != ""},
		ImageUrl:   req.ImageURL,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(banner)
}

func (h *BannerHandler) DeleteBanner(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = h.Store.DeleteBanner(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
