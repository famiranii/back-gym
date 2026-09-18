package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type OrderHandler struct {
	store *db.Store
	sms   *sms.Client
}

func NewOrderHandler(store *db.Store, smsClient *sms.Client) *OrderHandler {
	return &OrderHandler{store: store, sms: smsClient}
}

func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)
	userID := payload.UserID

	var req struct {
		AddressID string `json:"address_id"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	addrID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid address_id"})
	}

	addr, err := h.store.GetUserAddress(c.Context(), db.GetUserAddressParams{
		ID:     addrID,
		UserID: userID,
	})
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "address not found"})
	}

	cartItems, err := h.store.GetCart(c.Context(), userID)
	if err != nil || len(cartItems) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cart is empty"})
	}

	// بررسی موجودی
	for _, item := range cartItems {
		variant, err := h.store.GetVariantByID(c.Context(), item.VariantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check stock"})
		}
		if int(variant.Stock) < int(item.Quantity) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "there isnt enough product",
				"product": item.ProductName,
			})
		}
	}

	shippingCost, err := h.store.GetShippingCost(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get shipping cost"})
	}

	var totalPrice int64
	for _, item := range cartItems {
		totalPrice += item.FinalPrice * int64(item.Quantity)
	}
	totalPrice += shippingCost

	order, err := h.store.CreateOrder(c.Context(), db.CreateOrderParams{
		UserID:            userID,
		Status:            "pending",
		ShippingCost:      shippingCost,
		TotalPrice:        totalPrice,
		AddressTitle:      addr.Title,
		AddressProvince:   addr.Province,
		AddressCity:       addr.City,
		AddressDetail:     addr.Address,
		AddressPostalCode: addr.PostalCode,
		AddressLat:        addr.Lat,
		AddressLng:        addr.Lng,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create order"})
	}

	for _, item := range cartItems {
		unitPrice := item.FinalPrice
		// خط 94 — CreateOrderItem
		_, err := h.store.CreateOrderItem(c.Context(), db.CreateOrderItemParams{
			OrderID:    order.ID,
			VariantID:  pgtype.UUID{Bytes: item.VariantID, Valid: true},
			ProductID:  pgtype.UUID{Bytes: item.ProductID, Valid: true},
			Quantity:   item.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: unitPrice * int64(item.Quantity),
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create order items"})
		}
	}

	h.store.ClearCart(c.Context(), userID)

	return c.Status(fiber.StatusCreated).JSON(order)
}
func (h *OrderHandler) GetMyOrders(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)
	userID := payload.UserID

	orders, err := h.store.GetOrdersByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get orders"})
	}
	return c.JSON(orders)
}
func (h *OrderHandler) GetUserOrders(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))

	orders, err := h.store.GetOrdersByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get orders"})
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

	order, err := h.store.GetOrderByID(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "order not found",
		})
	}

	items, err := h.store.GetOrderItems(c.Context(), orderID)
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order id",
		})
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}

	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid status",
		})
	}

	// فقط برای status های غیر paid
	if req.Status != "paid" {
		order, err := h.store.UpdateOrderStatus(
			c.Context(),
			db.UpdateOrderStatusParams{
				Status: req.Status,
				ID:     orderID,
			},
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update order status",
			})
		}

		return c.JSON(order)
	}

	// -----------------------------
	// paid transaction
	// -----------------------------

	var paidOrder db.Order

	err = h.store.ExecTx(c.Context(), func(q *db.Queries) error {
		order, err := q.MarkOrderAsPaid(c.Context(), orderID)
		if err != nil {
			return err
		}

		paidOrder = order

		items, err := q.GetOrderItems(c.Context(), orderID)
		if err != nil {
			return err
		}

		for _, item := range items {
			if !item.VariantID.Valid {
				continue
			}

			_, err := q.DecreaseVariantStock(
				c.Context(),
				db.DecreaseVariantStockParams{
					ID:    item.VariantID.Bytes,
					Stock: item.Quantity,
				},
			)

			if err != nil {
				return fmt.Errorf(
					"not enough stock for product %s: %w",
					item.ProductID,
					err,
				)
			}
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "not enough stock",
		})
	}

	// -----------------------------
	// SMS بعد از COMMIT
	// -----------------------------

	payload := c.Locals("payload").(*token.Payload)

	if h.sms != nil {
		msg := fmt.Sprintf(
			"سفارش شما با موفقیت پرداخت شد.\nمبلغ کل: %d تومان\nکد پیگیری: %s",
			paidOrder.TotalPrice,
			paidOrder.ID.String()[:8],
		)

		go func(phone, message string) {
			if _, err := h.sms.Send(
				context.Background(),
				[]string{phone},
				message,
				false,
			); err != nil {
				log.Printf("failed to send order confirmation sms: %v", err)
			}
		}(payload.Phone, msg)
	}

	return c.JSON(paidOrder)
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

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid offset",
		})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
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

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid offset",
		})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit <= 0 || limit > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid limit",
		})
	}

	// فیلتر تاریخ — اختیاری
	var fromTime, toTime pgtype.Timestamptz

	if date := c.Query("date"); date != "" {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid date, use YYYY-MM-DD",
			})
		}

		// شروع روز
		from := time.Date(
			t.Year(),
			t.Month(),
			t.Day(),
			0, 0, 0, 0,
			t.Location(),
		)

		// شروع روز بعد
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

	// جستجوی شناسه سفارش — اختیاری
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
		fmt.Println("GetAdminOrdersByStatus ERROR:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get orders",
		})
	}

	if orders == nil {
		orders = []db.Order{}
	}

	return c.JSON(orders)
}
