package handlers

import (
	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/gofiber/fiber/v3"
)

type AdminHandler struct {
	store *db.Store
}

func NewAdminHandler(store *db.Store) *AdminHandler {
	return &AdminHandler{
		store: store,
	}
}

func (h *AdminHandler) GetDashboard(c fiber.Ctx) error {
	ctx := c.Context()

	// stats
	todayOrders, err := h.store.GetTodayOrdersCount(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get today orders"},
		)
	}

	todaySales, err := h.store.GetTodaySales(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get today sales"},
		)
	}

	pendingOrders, err := h.store.GetPendingOrdersCount(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get pending orders"},
		)
	}

	paidOrders, err := h.store.GetPaidOrdersCount(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get paid orders"},
		)
	}

	// sales chart
	sales, err := h.store.GetSalesLast7Days(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get sales"},
		)
	}

	// recent orders
	recentOrders, err := h.store.GetRecentOrders(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get recent orders"},
		)
	}

	// low stock
	lowStock, err := h.store.GetLowStockProducts(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get low stock products"},
		)
	}

	// top products
	topProducts, err := h.store.GetTopProducts(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get top products"},
		)
	}

	// shipping cost
	shippingCost, err := h.store.GetShippingCost(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to get shipping cost"},
		)
	}

	return c.JSON(fiber.Map{
		"stats": fiber.Map{
			"today_orders":   todayOrders,
			"today_sales":    todaySales,
			"pending_orders": pendingOrders,
			"paid_orders":    paidOrders,
		},

		"sales": sales,

		"recent_orders": recentOrders,

		"low_stock_products": lowStock,

		"top_products": topProducts,

		"shipping_cost": shippingCost,
	})
}
