package handlers

import (
	"strconv"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderHandler struct {
	store *db.Store
}

func NewOrderHandler(store *db.Store) *OrderHandler {
	return &OrderHandler{store: store}
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order id"})
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"shipped":   true,
		"delivered": true,
		"cancelled": true,
	}
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	order, err := h.store.UpdateOrderStatus(c.Context(), db.UpdateOrderStatusParams{
		Status: req.Status,
		ID:     orderID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update order status"})
	}

	// کم کردن موجودی وقتی paid میشه
	if req.Status == "paid" {
		items, err := h.store.GetOrderItems(c.Context(), orderID)
		if err == nil {
			for _, item := range items {
				if item.VariantID.Valid {
					h.store.DecreaseVariantStock(c.Context(), db.DecreaseVariantStockParams{
						ID:    item.VariantID.Bytes,
						Stock: item.Quantity,
					})
				}
			}
		}
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
