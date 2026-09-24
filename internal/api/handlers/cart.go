// internal/handlers/cart_handler.go

package handlers

import (
	"encoding/json"
	"net/url"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type CartHandler struct {
	Store *db.Store
}

func NewCartHandler(store *db.Store) *CartHandler {
	return &CartHandler{Store: store}
}

// =====================================================
// Types
// =====================================================

type AddToCartRequest struct {
	VariantID string `json:"variant_id" validate:"required"`
	Quantity  int32  `json:"quantity" validate:"required,gt=0"`
}

type UpdateCartRequest struct {
	Quantity int32 `json:"quantity" validate:"required,gt=0"`
}

type GuestCartItem struct {
	VariantID string `json:"variant_id"`
	Quantity  int32  `json:"quantity"`
}

// =====================================================
// GET /cart
// =====================================================

func (h *CartHandler) GetCart(c fiber.Ctx) error {

	// -------------------------------------------------
	// Logged-in user
	// -------------------------------------------------

	if payloadValue := c.Locals("payload"); payloadValue != nil {

		payload, ok := payloadValue.(*token.Payload)

		if ok {
			user, err := h.Store.GetUserByID(
				c.Context(),
				payload.UserID,
			)

			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(
					fiber.Map{"error": "user not found"},
				)
			}

			items, err := h.Store.GetCart(
				c.Context(),
				user.ID,
			)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					fiber.Map{"error": err.Error()},
				)
			}

			// همان ساختار اصلی
			return c.JSON(items)
		}
	}

	// -------------------------------------------------
	// Guest user
	// -------------------------------------------------

	cookie := c.Cookies("guest_cart")

	if cookie == "" {
		return c.JSON([]db.GetCartRow{})
	}

	decoded, err := url.QueryUnescape(cookie)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid guest cart"},
		)
	}

	var guestItems []GuestCartItem

	if err := json.Unmarshal(
		[]byte(decoded),
		&guestItems,
	); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid guest cart"},
		)
	}

	if len(guestItems) == 0 {
		return c.JSON([]db.GetCartRow{})
	}

	variantIDs := make([]uuid.UUID, 0, len(guestItems))

	quantities := make(map[uuid.UUID]int32)

	for _, item := range guestItems {

		variantID, err := uuid.Parse(item.VariantID)
		if err != nil {
			continue
		}

		variantIDs = append(variantIDs, variantID)
		quantities[variantID] = item.Quantity
	}

	if len(variantIDs) == 0 {
		return c.JSON([]db.GetCartRow{})
	}

	guestProducts, err := h.Store.GetGuestCart(
		c.Context(),
		variantIDs,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	// تبدیل به دقیقاً همان ساختار GetCartRow
	items := make([]db.GetCartRow, 0, len(guestProducts))

	for _, item := range guestProducts {

		items = append(items, db.GetCartRow{
			// Guest cart هنوز cart_items.id ندارد
			// بنابراین VariantID را به عنوان ID استفاده می‌کنیم.
			ID:          item.VariantID,
			Quantity:    quantities[item.VariantID],
			VariantID:   item.VariantID,
			Label:       item.Label,
			Color:       item.Color,
			Stock:       item.Stock,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Price:       item.Price,
			Discount:    item.Discount,
			FinalPrice:  item.FinalPrice,
			ImageUrl:    item.ImageUrl,
		})
	}

	return c.JSON(items)
}

// =====================================================
// POST /cart
// =====================================================

func (h *CartHandler) AddToCart(c fiber.Ctx) error {

	var req AddToCartRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	variantID, err := uuid.Parse(req.VariantID)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid variant_id"},
		)
	}

	// -------------------------------------------------
	// Logged-in user
	// -------------------------------------------------

	if payloadValue := c.Locals("payload"); payloadValue != nil {

		payload, ok := payloadValue.(*token.Payload)

		if ok {
			user, err := h.Store.GetUserByID(
				c.Context(),
				payload.UserID,
			)

			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(
					fiber.Map{"error": "user not found"},
				)
			}

			item, err := h.Store.AddToCart(
				c.Context(),
				db.AddToCartParams{
					UserID:    user.ID,
					VariantID: variantID,
					Quantity:  req.Quantity,
				},
			)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					fiber.Map{"error": err.Error()},
				)
			}

			return c.Status(fiber.StatusOK).JSON(item)
		}
	}

	// -------------------------------------------------
	// Guest
	// -------------------------------------------------

	return h.addToGuestCart(c, req)
}

// =====================================================
// Add Guest Cart
// =====================================================

func (h *CartHandler) addToGuestCart(
	c fiber.Ctx,
	req AddToCartRequest,
) error {

	cart, err := h.getGuestCartCookie(c)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid guest cart"},
		)
	}

	found := false

	for i := range cart {

		if cart[i].VariantID == req.VariantID {

			cart[i].Quantity += req.Quantity
			found = true

			break
		}
	}

	if !found {
		cart = append(cart, GuestCartItem{
			VariantID: req.VariantID,
			Quantity:  req.Quantity,
		})
	}

	if err := h.setGuestCartCookie(c, cart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to save guest cart"},
		)
	}

	// بعد از اضافه شدن، همان ساختار cart را از GET بگیر
	// تا پاسخ POST هم با GET هماهنگ باشد.
	return h.GetCart(c)
}

// =====================================================
// PATCH /cart/:id
// =====================================================

func (h *CartHandler) UpdateCartItem(c fiber.Ctx) error {

	var req UpdateCartRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	// -------------------------------------------------
	// Logged-in user
	// -------------------------------------------------

	if payloadValue := c.Locals("payload"); payloadValue != nil {

		payload, ok := payloadValue.(*token.Payload)

		if ok {
			user, err := h.Store.GetUserByID(
				c.Context(),
				payload.UserID,
			)

			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(
					fiber.Map{"error": "user not found"},
				)
			}

			itemID, err := uuid.Parse(c.Params("id"))

			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(
					fiber.Map{"error": "invalid id"},
				)
			}

			_, err = h.Store.UpdateCartItemQuantity(
				c.Context(),
				db.UpdateCartItemQuantityParams{
					ID:       itemID,
					UserID:   user.ID,
					Quantity: req.Quantity,
				},
			)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					fiber.Map{"error": err.Error()},
				)
			}

			// خروجی همان GET /cart
			return h.GetCart(c)
		}
	}

	// -------------------------------------------------
	// Guest
	// -------------------------------------------------

	variantID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid id"},
		)
	}

	cart, err := h.getGuestCartCookie(c)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid guest cart"},
		)
	}

	found := false

	for i := range cart {

		if cart[i].VariantID == variantID.String() {

			cart[i].Quantity = req.Quantity
			found = true

			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": "cart item not found"},
		)
	}

	if err := h.setGuestCartCookie(c, cart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to update guest cart"},
		)
	}

	return h.GetCart(c)
}

// =====================================================
// DELETE /cart/:id
// =====================================================

func (h *CartHandler) RemoveFromCart(c fiber.Ctx) error {

	// -------------------------------------------------
	// Logged-in user
	// -------------------------------------------------

	if payloadValue := c.Locals("payload"); payloadValue != nil {

		payload, ok := payloadValue.(*token.Payload)

		if ok {
			user, err := h.Store.GetUserByID(
				c.Context(),
				payload.UserID,
			)

			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(
					fiber.Map{"error": "user not found"},
				)
			}

			itemID, err := uuid.Parse(c.Params("id"))

			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(
					fiber.Map{"error": "invalid id"},
				)
			}

			err = h.Store.RemoveFromCart(
				c.Context(),
				db.RemoveFromCartParams{
					ID:     itemID,
					UserID: user.ID,
				},
			)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					fiber.Map{"error": err.Error()},
				)
			}

			return h.GetCart(c)
		}
	}

	// -------------------------------------------------
	// Guest
	// -------------------------------------------------

	variantID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid id"},
		)
	}

	cart, err := h.getGuestCartCookie(c)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid guest cart"},
		)
	}

	newCart := make([]GuestCartItem, 0, len(cart))

	found := false

	for _, item := range cart {

		if item.VariantID == variantID.String() {
			found = true
			continue
		}

		newCart = append(newCart, item)
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": "cart item not found"},
		)
	}

	if err := h.setGuestCartCookie(c, newCart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to update guest cart"},
		)
	}

	return h.GetCart(c)
}

// =====================================================
// DELETE /cart
// =====================================================

func (h *CartHandler) ClearCart(c fiber.Ctx) error {

	// -------------------------------------------------
	// Logged-in user
	// -------------------------------------------------

	if payloadValue := c.Locals("payload"); payloadValue != nil {

		payload, ok := payloadValue.(*token.Payload)

		if ok {
			user, err := h.Store.GetUserByID(
				c.Context(),
				payload.UserID,
			)

			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(
					fiber.Map{"error": "user not found"},
				)
			}

			err = h.Store.ClearCart(
				c.Context(),
				user.ID,
			)

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					fiber.Map{"error": err.Error()},
				)
			}

			return c.JSON([]db.GetCartRow{})
		}
	}

	// -------------------------------------------------
	// Guest
	// -------------------------------------------------

	c.Cookie(&fiber.Cookie{
		Name:     "guest_cart",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "none",
		MaxAge:   -1,
	})

	return c.JSON([]db.GetCartRow{})
}

// =====================================================
// Guest Cart Cookie Helpers
// =====================================================

func (h *CartHandler) getGuestCartCookie(
	c fiber.Ctx,
) ([]GuestCartItem, error) {

	cookie := c.Cookies("guest_cart")

	if cookie == "" {
		return []GuestCartItem{}, nil
	}

	decoded, err := url.QueryUnescape(cookie)

	if err != nil {
		return nil, err
	}

	var cart []GuestCartItem

	if err := json.Unmarshal(
		[]byte(decoded),
		&cart,
	); err != nil {
		return nil, err
	}

	return cart, nil
}

func (h *CartHandler) setGuestCartCookie(
	c fiber.Ctx,
	cart []GuestCartItem,
) error {

	data, err := json.Marshal(cart)

	if err != nil {
		return err
	}

	encoded := url.QueryEscape(string(data))

	c.Cookie(&fiber.Cookie{
		Name:     "guest_cart",
		Value:    encoded,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "none",
		MaxAge:   60 * 60 * 24 * 30,
	})

	return nil
}
