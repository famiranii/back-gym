package handlers

import (
	"fmt"
	"strconv"
	"time"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderHandler struct {
	store *db.Store
	sms   *sms.Client
}

func NewOrderHandler(store *db.Store, smsClient *sms.Client) *OrderHandler {
	return &OrderHandler{
		store: store,
		sms:   smsClient,
	}
}

func calculateDiscount(discount db.DiscountCode, subtotal int64) int64 {
	if subtotal < discount.MinOrderAmount {
		return 0
	}

	var amount int64

	switch discount.DiscountType {
	case "percentage":
		amount = subtotal * discount.DiscountValue / 100

	case "fixed":
		amount = discount.DiscountValue
	}

	if discount.MaxDiscountAmount.Valid &&
		discount.MaxDiscountAmount.Int64 > 0 &&
		amount > discount.MaxDiscountAmount.Int64 {
		amount = discount.MaxDiscountAmount.Int64
	}

	if amount > subtotal {
		amount = subtotal
	}

	return amount
}

func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)
	userID := payload.UserID

	var req struct {
		AddressID    string `json:"address_id"`
		DiscountCode string `json:"discount_code"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	// -----------------------------
	// address
	// -----------------------------

	addrID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid address_id",
		})
	}

	addr, err := h.store.GetUserAddress(
		c.Context(),
		db.GetUserAddressParams{
			ID:     addrID,
			UserID: userID,
		},
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "address not found",
		})
	}

	// -----------------------------
	// cart
	// -----------------------------

	cartItems, err := h.store.GetCart(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get cart",
		})
	}

	if len(cartItems) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cart is empty",
		})
	}

	// -----------------------------
	// بررسی موجودی
	// -----------------------------

	for _, item := range cartItems {
		variant, err := h.store.GetVariantByID(
			c.Context(),
			item.VariantID,
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check stock",
			})
		}

		if int(variant.Stock) < int(item.Quantity) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "there isnt enough product",
				"product": item.ProductName,
			})
		}
	}

	// -----------------------------
	// shipping
	// -----------------------------

	shippingCost, err := h.store.GetShippingCost(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get shipping cost",
		})
	}

	// -----------------------------
	// subtotal
	// -----------------------------

	var subtotal int64

	for _, item := range cartItems {
		subtotal += item.FinalPrice * int64(item.Quantity)
	}

	// -----------------------------
	// discount
	// -----------------------------

	var discountAmount int64
	var discountCode string

	if req.DiscountCode != "" {
		discount, err := h.store.GetValidDiscountCode(
			c.Context(),
			req.DiscountCode,
		)

		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid discount code",
			})
		}

		if subtotal < discount.MinOrderAmount {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "minimum order amount not reached",
			})
		}

		discountAmount = calculateDiscount(
			discount,
			subtotal,
		)

		discountCode = discount.Code
	}

	// -----------------------------
	// total
	// -----------------------------

	totalPrice := subtotal - discountAmount + shippingCost

	// -----------------------------
	// create order
	// -----------------------------

	order, err := h.store.CreateOrder(
		c.Context(),
		db.CreateOrderParams{
			UserID:       userID,
			Status:       "pending",
			ShippingCost: shippingCost,
			TotalPrice:   totalPrice,
			DiscountCode: pgtype.Text{
				String: discountCode,
				Valid:  discountCode != "",
			},
			DiscountAmount:    discountAmount,
			AddressTitle:      addr.Title,
			AddressProvince:   addr.Province,
			AddressCity:       addr.City,
			AddressDetail:     addr.Address,
			AddressPostalCode: addr.PostalCode,
			AddressLat:        addr.Lat,
			AddressLng:        addr.Lng,
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create order",
		})
	}

	// -----------------------------
	// create order items
	// -----------------------------

	for _, item := range cartItems {
		unitPrice := item.FinalPrice

		_, err := h.store.CreateOrderItem(
			c.Context(),
			db.CreateOrderItemParams{
				OrderID: order.ID,
				VariantID: pgtype.UUID{
					Bytes: item.VariantID,
					Valid: true,
				},
				ProductID: pgtype.UUID{
					Bytes: item.ProductID,
					Valid: true,
				},
				Quantity:   item.Quantity,
				UnitPrice:  unitPrice,
				TotalPrice: unitPrice * int64(item.Quantity),
			},
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create order items",
			})
		}
	}

	// -----------------------------
	// clear cart
	// -----------------------------

	if err := h.store.ClearCart(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to clear cart",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *OrderHandler) GetMyOrders(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)
	userID := payload.UserID

	// سفارش‌های pending بیشتر از ۱۵ دقیقه را لغو کن
	if err := h.store.CancelExpiredOrdersByPaymentForUser(
		c.Context(),
		userID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "failed to cleanup expired orders",
			},
		)
	}

	orders, err := h.store.GetOrdersByUser(
		c.Context(),
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "failed to get orders",
			},
		)
	}

	if orders == nil {
		orders = []db.Order{}
	}

	return c.JSON(orders)
}
func (h *OrderHandler) GetUserOrders(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	orders, err := h.store.GetOrdersByUser(
		c.Context(),
		userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get orders",
		})
	}

	return c.JSON(orders)
}

func (h *OrderHandler) GetOrderDetail(c fiber.Ctx) error {
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order id",
		})
	}

	order, err := h.store.GetOrderByID(
		c.Context(),
		orderID,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "order not found",
		})
	}

	items, err := h.store.GetOrderItems(
		c.Context(),
		orderID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get order items",
		})
	}

	return c.JSON(fiber.Map{
		"order": order,
		"items": items,
	})
}

func (h *OrderHandler) UpdateOrderStatus(c fiber.Ctx) error {

	orderID, err := uuid.Parse(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid order id",
			},
		)
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid request",
			},
		)
	}

	// --------------------------------------------------------
	// Valid statuses
	// --------------------------------------------------------

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}

	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid status",
			},
		)
	}

	// --------------------------------------------------------
	// paid فقط از طریق پرداخت ثبت می‌شود
	// --------------------------------------------------------

	if req.Status == "paid" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "paid status can only be set by payment",
			},
		)
	}

	// --------------------------------------------------------
	// Update status
	// --------------------------------------------------------

	order, err := h.store.UpdateOrderStatus(
		c.Context(),
		db.UpdateOrderStatusParams{
			Status: req.Status,
			ID:     orderID,
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "failed to update order status",
			},
		)
	}

	return c.JSON(order)
}

func (h *OrderHandler) GetOrdersByStatus(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)
	userID := payload.UserID

	status := c.Params("status")

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}

	if !validStatuses[status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid status",
		})
	}

	offset, err := strconv.Atoi(
		c.Query("offset", "0"),
	)
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid offset",
		})
	}

	limit, err := strconv.Atoi(
		c.Query("limit", "10"),
	)
	if err != nil || limit <= 0 || limit > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid limit",
		})
	}

	orders, err := h.store.GetOrdersByUserAndStatus(
		c.Context(),
		db.GetOrdersByUserAndStatusParams{
			UserID: userID,
			Status: status,
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get orders",
		})
	}

	if orders == nil {
		orders = []db.Order{}
	}

	return c.JSON(orders)
}

func (h *OrderHandler) GetAdminOrdersByStatus(c fiber.Ctx) error {
	status := c.Params("status")

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}

	if !validStatuses[status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid status",
		})
	}

	offset, err := strconv.Atoi(
		c.Query("offset", "0"),
	)
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid offset",
		})
	}

	limit, err := strconv.Atoi(
		c.Query("limit", "10"),
	)
	if err != nil || limit <= 0 || limit > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid limit",
		})
	}

	// -----------------------------
	// فیلتر تاریخ
	// -----------------------------

	var fromTime, toTime pgtype.Timestamptz

	if date := c.Query("date"); date != "" {
		t, err := time.Parse(
			"2006-01-02",
			date,
		)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid date, use YYYY-MM-DD",
			})
		}

		from := time.Date(
			t.Year(),
			t.Month(),
			t.Day(),
			0,
			0,
			0,
			0,
			t.Location(),
		)

		to := from.AddDate(0, 0, 1)

		fromTime = pgtype.Timestamptz{
			Time:  from,
			Valid: true,
		}

		toTime = pgtype.Timestamptz{
			Time:  to,
			Valid: true,
		}
	}

	// -----------------------------
	// search
	// -----------------------------

	search := c.Query("search", "")

	orders, err := h.store.GetAdminOrdersByStatus(
		c.Context(),
		db.GetAdminOrdersByStatusParams{
			Status:  status,
			Column2: fromTime,
			Column3: toTime,
			Column4: search,
			Limit:   int32(limit),
			Offset:  int32(offset),
		},
	)

	if err != nil {
		fmt.Println(
			"GetAdminOrdersByStatus ERROR:",
			err,
		)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get orders",
		})
	}

	if orders == nil {
		orders = []db.Order{}
	}

	return c.JSON(orders)
}
