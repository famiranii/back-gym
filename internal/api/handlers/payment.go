package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/payment"
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PaymentHandler struct {
	Store    *db.Store
	Config   util.Config
	ZarinPal *payment.ZarinPalService
	SMS      *sms.Client
}

func NewPaymentHandler(
	store *db.Store,
	config util.Config,
	zarinPal *payment.ZarinPalService,
	smsClient *sms.Client,
) *PaymentHandler {
	return &PaymentHandler{
		Store:    store,
		Config:   config,
		ZarinPal: zarinPal,
		SMS:      smsClient,
	}
}

// ============================================================
// Request Payment
// POST /api/payment/zarinpal/request
// ============================================================

type CreatePaymentRequest struct {
	OrderID string `json:"order_id"`
}

func (h *PaymentHandler) CreateZarinPalPayment(c fiber.Ctx) error {

	// --------------------------------------------------------
	// Auth
	// --------------------------------------------------------

	payload, ok := c.Locals("payload").(*token.Payload)
	if !ok || payload == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{
				"error": "unauthorized",
			},
		)
	}

	// --------------------------------------------------------
	// Request body
	// --------------------------------------------------------

	var req CreatePaymentRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid request body",
			},
		)
	}

	// --------------------------------------------------------
	// Parse UUID
	// --------------------------------------------------------

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid order id",
			},
		)
	}

	// --------------------------------------------------------
	// Get order
	// --------------------------------------------------------

	order, err := h.Store.GetOrderByID(
		c.Context(),
		orderID,
	)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{
				"error": "order not found",
			},
		)
	}

	// --------------------------------------------------------
	// Security
	// --------------------------------------------------------

	if order.UserID != payload.UserID {
		return c.Status(fiber.StatusForbidden).JSON(
			fiber.Map{
				"error": "you cannot pay this order",
			},
		)
	}

	// --------------------------------------------------------
	// فقط سفارش pending قابل پرداخت است.
	// --------------------------------------------------------

	if order.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "order is not payable",
			},
		)
	}

	// --------------------------------------------------------
	// مبلغ از DB
	// --------------------------------------------------------

	amount := order.TotalPrice

	if amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "invalid order amount",
			},
		)
	}

	// --------------------------------------------------------
	// Request ZarinPal
	// --------------------------------------------------------

	authority, err := h.ZarinPal.RequestPayment(
		c.Context(),
		amount,
		fmt.Sprintf(
			"پرداخت سفارش %s",
			order.ID.String(),
		),
		order.ID.String(),
		"",
		"",
	)

	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(
			fiber.Map{
				"error": err.Error(),
			},
		)
	}

	// --------------------------------------------------------
	// Save Payment
	// --------------------------------------------------------

	expiresAt := time.Now().Add(15 * time.Minute)

	paymentRecord, err := h.Store.CreatePayment(
		c.Context(),
		db.CreatePaymentParams{
			OrderID:   order.ID,
			Amount:    amount,
			Authority: authority,
			ExpiresAt: pgtype.Timestamptz{
				Time:  expiresAt,
				Valid: true,
			},
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "failed to create payment",
			},
		)
	}

	// --------------------------------------------------------
	// Response
	// --------------------------------------------------------

	return c.Status(fiber.StatusCreated).JSON(
		fiber.Map{
			"payment_id": paymentRecord.ID,
			"order_id":   order.ID,
			"authority":  authority,
			"payment_url": h.ZarinPal.PaymentURL(
				authority,
			),
			"expires_at": expiresAt,
		},
	)
}

// ============================================================
// ZarinPal Callback
// GET /api/payment/zarinpal/callback
// ============================================================

func (h *PaymentHandler) ZarinPalCallback(c fiber.Ctx) error {

	authority := c.Query("Authority")
	status := c.Query("Status")

	if authority == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "authority is missing",
			},
		)
	}

	// --------------------------------------------------------
	// Get Payment
	// --------------------------------------------------------

	paymentRecord, err := h.Store.GetPaymentByAuthority(
		c.Context(),
		authority,
	)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{
				"error": "payment not found",
			},
		)
	}

	// --------------------------------------------------------
	// Payment expiration
	// --------------------------------------------------------

	if !paymentRecord.ExpiresAt.Valid {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "payment expiration is invalid",
			},
		)
	}

	if time.Now().After(paymentRecord.ExpiresAt.Time) {

		_, _ = h.Store.MarkPaymentExpired(
			c.Context(),
			paymentRecord.ID,
		)

		return c.Redirect().To(
			fmt.Sprintf(
				"%s/checkout/failed?reason=expired",
				h.Config.FRONTEND_URL,
			),
		)
	}

	// --------------------------------------------------------
	// Already paid
	// --------------------------------------------------------

	if paymentRecord.Status == "paid" {
		return c.Redirect().To(
			fmt.Sprintf(
				"%s/checkout/success?payment_id=%d",
				h.Config.FRONTEND_URL,
				paymentRecord.ID,
			),
		)
	}

	// --------------------------------------------------------
	// User cancelled payment
	// --------------------------------------------------------

	if status != "OK" {

		_, _ = h.Store.MarkPaymentFailed(
			c.Context(),
			paymentRecord.ID,
		)

		return c.Redirect().To(
			fmt.Sprintf(
				"%s/checkout/failed",
				h.Config.FRONTEND_URL,
			),
		)
	}

	// --------------------------------------------------------
	// Verify with ZarinPal
	// --------------------------------------------------------

	result, err := h.ZarinPal.VerifyPayment(
		c.Context(),
		paymentRecord.Amount,
		paymentRecord.Authority,
	)

	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(
			fiber.Map{
				"error": err.Error(),
			},
		)
	}

	// --------------------------------------------------------
	// Payment result
	// --------------------------------------------------------

	switch result.Data.Code {

	case 100, 101:

		// ----------------------------------------------------
		// پرداخت موفق:
		//
		// 1. Payment -> paid
		// 2. Order -> paid
		// 3. Stock -> decrease
		//
		// همه داخل یک transaction
		// ----------------------------------------------------

		paidOrder, err := h.finalizeSuccessfulPayment(
			c,
			paymentRecord,
			result.Data.RefID,
			result.Data.CardPan,
			result.Data.CardHash,
			result.Data.FeeType,
			result.Data.Fee,
		)

		if err != nil {
			log.Printf(
				"ZARINPAL FINALIZE ERROR: payment_id=%d order_id=%s error=%v",
				paymentRecord.ID,
				paymentRecord.OrderID.String(),
				err,
			)

			return c.Status(
				fiber.StatusInternalServerError,
			).JSON(
				fiber.Map{
					"error": err.Error(),
				},
			)
		}

		// ----------------------------------------------------
		// SMS
		//
		// این قسمت بعد از commit شدن transaction اجرا می‌شود.
		// بنابراین خراب شدن SMS باعث rollback پرداخت نمی‌شود.
		// ----------------------------------------------------

		if h.SMS != nil {

			// شماره موبایل صاحب سفارش را از DB می‌گیریم.
			//
			// callback زرین‌پال لزوماً JWT ندارد،
			// بنابراین از c.Locals("payload") استفاده نمی‌کنیم.

			user, userErr := h.Store.GetUserByID(
				context.Background(),
				paidOrder.UserID,
			)

			if userErr != nil {

				log.Printf(
					"ORDER SMS USER ERROR: order_id=%s user_id=%s error=%v",
					paidOrder.ID.String(),
					paidOrder.UserID.String(),
					userErr,
				)

			} else if user.Phone != "" {

				message := fmt.Sprintf(
					"سفارش شما با موفقیت پرداخت شد.\nمبلغ کل: %d تومان\nکد پیگیری: %s",
					paidOrder.TotalPrice,
					paidOrder.ID.String()[:8],
				)

				go func(
					phone string,
					message string,
					orderID uuid.UUID,
				) {

					_, smsErr := h.SMS.Send(
						context.Background(),
						[]string{phone},
						message,
						false,
					)

					if smsErr != nil {
						log.Printf(
							"ORDER SMS ERROR: order_id=%s phone=%q error=%v",
							orderID.String(),
							phone,
							smsErr,
						)

						return
					}

					log.Printf(
						"ORDER SMS SENT: order_id=%s phone=%q",
						orderID.String(),
						phone,
					)

				}(
					user.Phone,
					message,
					paidOrder.ID,
				)

			} else {

				log.Printf(
					"ORDER SMS SKIPPED: user phone is empty, order_id=%s user_id=%s",
					paidOrder.ID.String(),
					paidOrder.UserID.String(),
				)
			}
		}

		// ----------------------------------------------------
		// این redirect قبلی خودت است.
		// تغییرش ندادیم.
		// ----------------------------------------------------

		return c.Redirect().To(
			fmt.Sprintf(
				"%s/checkout/success?ref_id=%d",
				h.Config.FRONTEND_URL,
				result.Data.RefID,
			),
		)

	default:

		_, _ = h.Store.MarkPaymentFailed(
			c.Context(),
			paymentRecord.ID,
		)

		return c.Redirect().To(
			fmt.Sprintf(
				"%s/checkout/failed",
				h.Config.FRONTEND_URL,
			),
		)
	}
}

// ============================================================
// Finalize Successful Payment
// ============================================================
//
// این تابع فقط وقتی پرداخت زرین‌پال موفق شده اجرا می‌شود.
//
// Transaction:
//
// Payment -> paid
// Order   -> paid
// Stock   -> decrease
//
// اگر هر کدام fail شوند، کل transaction rollback می‌شود.
// ============================================================

func (h *PaymentHandler) finalizeSuccessfulPayment(
	c fiber.Ctx,
	paymentRecord db.Payment,
	refID int64,
	cardPan string,
	cardHash string,
	feeType string,
	fee int64,
) (db.Order, error) {

	var paidOrder db.Order

	err := h.Store.ExecTx(
		c.Context(),
		func(q *db.Queries) error {

			// ------------------------------------------------
			// Payment -> Paid
			// ------------------------------------------------

			_, err := q.MarkPaymentPaid(
				c.Context(),
				db.MarkPaymentPaidParams{
					ID: paymentRecord.ID,

					RefID: pgtype.Int8{
						Int64: refID,
						Valid: true,
					},

					CardPan: pgtype.Text{
						String: cardPan,
						Valid:  cardPan != "",
					},

					CardHash: pgtype.Text{
						String: cardHash,
						Valid:  cardHash != "",
					},

					FeeType: pgtype.Text{
						String: feeType,
						Valid:  feeType != "",
					},

					Fee: pgtype.Int8{
						Int64: fee,
						Valid: true,
					},
				},
			)

			if err != nil {
				return fmt.Errorf(
					"failed to mark payment as paid: %w",
					err,
				)
			}

			// ------------------------------------------------
			// Order -> Paid
			// ------------------------------------------------

			paidOrder, err = q.MarkOrderAsPaid(
				c.Context(),
				paymentRecord.OrderID,
			)

			if err != nil {
				return fmt.Errorf(
					"failed to mark order as paid: %w",
					err,
				)
			}

			// ------------------------------------------------
			// Get Order Items
			// ------------------------------------------------

			items, err := q.GetOrderItems(
				c.Context(),
				paymentRecord.OrderID,
			)

			if err != nil {
				return fmt.Errorf(
					"failed to get order items: %w",
					err,
				)
			}

			// ------------------------------------------------
			// Decrease Stock
			// ------------------------------------------------

			for _, item := range items {

				if !item.VariantID.Valid {
					continue
				}

				if item.Quantity <= 0 {
					return fmt.Errorf(
						"invalid item quantity: order_item_id=%s quantity=%d",
						item.ID.String(),
						item.Quantity,
					)
				}

				variantID := uuid.UUID(item.VariantID.Bytes)

				_, err := q.DecreaseVariantStock(
					c.Context(),
					db.DecreaseVariantStockParams{
						ID:    variantID,
						Stock: item.Quantity,
					},
				)

				if err != nil {
					return fmt.Errorf(
						"failed to decrease stock: variant_id=%s quantity=%d: %w",
						variantID.String(),
						item.Quantity,
						err,
					)
				}
			}

			return nil
		},
	)

	if err != nil {
		return db.Order{}, err
	}

	return paidOrder, nil
}
