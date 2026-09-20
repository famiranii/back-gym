package handlers

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/sms"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

const (
	otpPurposeRegister = "register"
	otpPurposeReset    = "reset_password"

	otpTTL         = 2 * time.Minute
	otpMaxAttempts = 5
	otpDigits      = 6

	// otpResendCooldown throttles resend requests so the same phone can't spam
	// SMS. A new code is only issued once this much time has passed since the
	// previous one was created.
	otpResendCooldown = 60 * time.Second
)

// OTPHandler owns the phone-verification flows: registration via OTP and
// password recovery via OTP.
type OTPHandler struct {
	Store      *db.Store
	SMS        *sms.Client
	Config     util.Config
	TokenMaker token.Maker
}

func NewOTPHandler(store *db.Store, smsClient *sms.Client, tokenMaker token.Maker, config util.Config) *OTPHandler {
	return &OTPHandler{
		Store:      store,
		SMS:        smsClient,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

// phoneRegexp mirrors the frontend registerSchema: Iranian 09xxxxxxxxx.
func normalizePhone(phone string) (string, bool) {
	p := strings.TrimSpace(phone)
	// accept 98... or +98... and convert to 0...
	p = strings.TrimPrefix(p, "+")
	if strings.HasPrefix(p, "98") && len(p) == 12 {
		p = "0" + p[2:]
	}
	if len(p) != 11 || !strings.HasPrefix(p, "09") {
		return "", false
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return p, true
}

// sendCode texts the OTP and returns the send error (nil on success) so callers
// can decide whether to surface it. On success it logs the provider message ID
// so delivery can be traced; on failure it logs the provider error.
func (h *OTPHandler) sendCode(c fiber.Ctx, phone, code, kind string) error {
	msg := fmt.Sprintf("کد تایید شما: %s\nاین کد تا ۲ دقیقه معتبر است.", code)
	results, err := h.SMS.Send(c.Context(), []string{phone}, msg, false)
	if err != nil {
		log.Error().Err(err).Str("phone", phone).Str("purpose", kind).Msg("failed to send otp sms")
		return err
	}
	if len(results) > 0 {
		log.Info().Int64("message_id", results[0].MessageID).Str("phone", phone).Str("purpose", kind).Msg("otp sms sent")
	}
	return nil
}

// codeSentResponse builds the success body for a code-request endpoint. In debug
// mode it echoes the code so the flow can be tested without a working SMS
// provider; in production the code is never returned.
func (h *OTPHandler) codeSentResponse(phone, code string) fiber.Map {
	resp := fiber.Map{"message": "code sent", "phone": phone}
	if h.Config.APP_DEBUG == "true" {
		resp["debug_code"] = code
	}
	return resp
}

// -----------------------------------------------------------------------------
// Register: request OTP
// -----------------------------------------------------------------------------

type RegisterOTPRequest struct {
	FullName string `json:"full_name" validate:"required,min=3"`
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// RequestRegisterOTP validates the signup data, stores it with a fresh OTP, and
// texts the code. The account is NOT created until the code is verified.
func (h *OTPHandler) RequestRegisterOTP(c fiber.Ctx) error {
	var req RegisterOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	phone, ok := normalizePhone(req.Phone)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	// reject if a user already exists with this phone
	if _, err := h.Store.GetUserByPhone(c.Context(), phone); err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "phone already registered"})
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	code, err := sms.GenerateOTP(otpDigits)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate code"})
	}

	// clear previous pending register codes for this phone
	_ = h.Store.DeleteOTPByPhonePurpose(c.Context(), db.DeleteOTPByPhonePurposeParams{
		Phone:   phone,
		Purpose: otpPurposeRegister,
	})

	_, err = h.Store.CreateOTP(c.Context(), db.CreateOTPParams{
		Phone:     phone,
		Code:      code,
		Purpose:   otpPurposeRegister,
		FullName:  pgtype.Text{String: req.FullName, Valid: true},
		Password:  pgtype.Text{String: hashedPassword, Valid: true},
		ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(otpTTL), Valid: true},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store code"})
	}

	if err := h.sendCode(c, phone, code, otpPurposeRegister); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to send verification code"})
	}

	return c.JSON(h.codeSentResponse(phone, code))
}

// -----------------------------------------------------------------------------
// Register: verify OTP -> create account + auto login
// -----------------------------------------------------------------------------

type VerifyRegisterRequest struct {
	Phone string `json:"phone" validate:"required"`
	Code  string `json:"code" validate:"required"`
}

func (h *OTPHandler) VerifyRegisterOTP(c fiber.Ctx) error {
	var req VerifyRegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	phone, ok := normalizePhone(req.Phone)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	otp, ok := h.validateOTP(c, phone, otpPurposeRegister, req.Code)
	if !ok {
		return nil // response already written
	}

	// make sure it wasn't registered in the meantime
	if _, err := h.Store.GetUserByPhone(c.Context(), phone); err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "phone already registered"})
	}

	user, err := h.Store.CreateUser(c.Context(), db.CreateUserParams{
		FullName: otp.FullName.String,
		Phone:    phone,
		Password: otp.Password.String,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	_ = h.Store.ConsumeOTP(c.Context(), otp.ID)

	return h.issueSession(c, user)
}

// -----------------------------------------------------------------------------
// Password recovery: request OTP
// -----------------------------------------------------------------------------

type ForgotPasswordRequest struct {
	Phone string `json:"phone" validate:"required"`
}

// RequestPasswordReset always answers 200 so it never leaks which phone numbers
// are registered. The SMS is only sent when the phone actually exists.
func (h *OTPHandler) RequestPasswordReset(c fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	phone, ok := normalizePhone(req.Phone)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	okResponse := fiber.Map{"message": "if the phone is registered, a code has been sent"}

	if _, err := h.Store.GetUserByPhone(c.Context(), phone); err != nil {
		// unknown phone: pretend success
		return c.JSON(okResponse)
	}

	code, err := sms.GenerateOTP(otpDigits)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate code"})
	}

	_ = h.Store.DeleteOTPByPhonePurpose(c.Context(), db.DeleteOTPByPhonePurposeParams{
		Phone:   phone,
		Purpose: otpPurposeReset,
	})

	_, err = h.Store.CreateOTP(c.Context(), db.CreateOTPParams{
		Phone:     phone,
		Code:      code,
		Purpose:   otpPurposeReset,
		ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(otpTTL), Valid: true},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store code"})
	}
	if err := h.sendCode(c, phone, code, otpPurposeReset); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(
			fiber.Map{"error": err.Error()},
		)
	}

	return c.JSON(okResponse)
}

// -----------------------------------------------------------------------------
// Password recovery: verify OTP + set new password
// -----------------------------------------------------------------------------

type ResetPasswordRequest struct {
	Phone       string `json:"phone" validate:"required"`
	Code        string `json:"code" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (h *OTPHandler) ResetPassword(c fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	phone, ok := normalizePhone(req.Phone)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	user, err := h.Store.GetUserByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid code"})
	}

	otp, ok := h.validateOTP(c, phone, otpPurposeReset, req.Code)
	if !ok {
		return nil // response already written
	}

	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	if _, err := h.Store.UpdateUserPassword(c.Context(), db.UpdateUserPasswordParams{
		ID:       user.ID,
		Password: hashedPassword,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update password"})
	}

	_ = h.Store.ConsumeOTP(c.Context(), otp.ID)

	return c.JSON(fiber.Map{"message": "password updated"})
}

// -----------------------------------------------------------------------------
// Resend OTP (works for both register and password-reset flows)
// -----------------------------------------------------------------------------

type ResendOTPRequest struct {
	Phone   string `json:"phone" validate:"required"`
	Purpose string `json:"purpose" validate:"required,oneof=register reset_password"`
}

// ResendOTP re-issues a code for a phone/purpose that already has a pending OTP.
// It reuses the stored data (full_name/password for register) so the client only
// needs to send the phone, and throttles by otpResendCooldown. Like the initial
// request-reset endpoint it does not leak whether the phone exists.
func (h *OTPHandler) ResendOTP(c fiber.Ctx) error {
	var req ResendOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	phone, ok := normalizePhone(req.Phone)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	okResponse := fiber.Map{"message": "code sent", "phone": phone}

	prev, err := h.Store.GetLatestOTP(c.Context(), db.GetLatestOTPParams{
		Phone:   phone,
		Purpose: req.Purpose,
	})
	if err != nil {
		// no pending request for this phone/purpose: pretend success so we don't
		// leak state, but there's nothing to resend.
		return c.JSON(okResponse)
	}

	// already verified/used — nothing to resend
	if prev.ConsumedAt.Valid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code already used"})
	}

	// throttle: enforce a cooldown between sends
	if time.Since(prev.CreatedAt.Time) < otpResendCooldown {
		wait := int((otpResendCooldown - time.Since(prev.CreatedAt.Time)).Seconds())
		if wait < 1 {
			wait = 1
		}
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":       "please wait before requesting a new code",
			"retry_after": wait,
		})
	}

	code, err := sms.GenerateOTP(otpDigits)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate code"})
	}

	// replace the old code, carrying over the register payload (nil-safe for reset)
	_ = h.Store.DeleteOTPByPhonePurpose(c.Context(), db.DeleteOTPByPhonePurposeParams{
		Phone:   phone,
		Purpose: req.Purpose,
	})

	if _, err := h.Store.CreateOTP(c.Context(), db.CreateOTPParams{
		Phone:     phone,
		Code:      code,
		Purpose:   req.Purpose,
		FullName:  prev.FullName,
		Password:  prev.Password,
		ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(otpTTL), Valid: true},
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store code"})
	}

	if err := h.sendCode(c, phone, code, req.Purpose); err != nil {
		if req.Purpose == otpPurposeRegister {
			return c.Status(fiber.StatusBadGateway).JSON(
				fiber.Map{"error": "failed to send verification code"},
			)
		}

		// reset_password intentionally stays generic
		return c.JSON(okResponse)
	}
	return c.JSON(h.codeSentResponse(phone, code))
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// validateOTP loads the latest code for phone/purpose and checks expiry, attempt
// count, consumption, and match. On failure it writes the HTTP response and
// returns ok=false; callers must stop and `return nil` (the response is already
// written). It must NOT signal failure through c.JSON's return value — that is
// nil on success, so an error sentinel built from it would never fire.
func (h *OTPHandler) validateOTP(c fiber.Ctx, phone, purpose, code string) (db.OtpCode, bool) {
	otp, err := h.Store.GetLatestOTP(c.Context(), db.GetLatestOTPParams{
		Phone:   phone,
		Purpose: purpose,
	})
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid code"})
		return db.OtpCode{}, false
	}

	if otp.ConsumedAt.Valid {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code already used"})
		return db.OtpCode{}, false
	}
	if time.Now().After(otp.ExpiresAt.Time) {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code expired"})
		return db.OtpCode{}, false
	}
	if otp.Attempts >= otpMaxAttempts {
		_ = c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many attempts"})
		return db.OtpCode{}, false
	}
	if otp.Code != strings.TrimSpace(code) {
		_ = h.Store.IncrementOTPAttempts(c.Context(), otp.ID)
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid code"})
		return db.OtpCode{}, false
	}

	return otp, true
}

// issueSession mints access + refresh tokens, persists a session, sets cookies,
// and returns the user — same shape as LoginUser so the frontend flow matches.
func (h *OTPHandler) issueSession(c fiber.Ctx, user db.User) error {
	accessToken, accessPayload, err := h.TokenMaker.CreateToken(
		user.Phone, user.ID, user.IsAdmin, h.Config.ACCESS_TOKEN_DURATION,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	refreshToken, refreshPayload, err := h.TokenMaker.CreateToken(
		user.Phone, user.ID, user.IsAdmin, h.Config.REFRESH_TOKEN_DURATION,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	session, err := h.Store.CreateSession(c.Context(), db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Phone:        user.Phone,
		RefreshToken: refreshToken,
		UserAgent: pgtype.Text{
			String: string(c.Request().Header.UserAgent()),
			Valid:  true,
		},
		ClientIp: func() *netip.Addr {
			addr, err := netip.ParseAddr(c.IP())
			if err != nil {
				return nil
			}
			return &addr
		}(),
		IsBlocked: false,
		ExpiresAt: pgtype.Timestamp{Time: refreshPayload.ExpiredAt, Valid: true},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  accessPayload.ExpiredAt,
		HTTPOnly: true,
		Secure:    true,
		SameSite: "Lax",
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  refreshPayload.ExpiredAt,
		HTTPOnly: true,
		Secure:    true,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.Status(fiber.StatusCreated).JSON(LoginUserResponse{
		SessionID: session.ID,
		User:      NewUserResponse(user),
	})
}
