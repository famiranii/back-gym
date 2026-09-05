package handlers

import (
	"net/netip"

	db "github.com/famiranii/back-gym.git/internal/db/sqlc"
	"github.com/famiranii/back-gym.git/internal/token"
	"github.com/famiranii/back-gym.git/internal/util"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserHandler struct {
	Store      *db.Store
	Config     util.Config
	TokenMaker token.Maker
}

type CreateUserRequest struct {
	FullName    string `json:"full_name" validate:"required"`
	PhoneNumber string `json:"phone" validate:"required"`
	Password    string `json:"password" validate:"required"`
}
type userResponse struct {
	ID        uuid.UUID        `json:"id"`
	FullName  string           `json:"full_name"`
	Phone     string           `json:"phone"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
}

func NewUserHandler(store *db.Store, tokenMaker token.Maker, config util.Config) *UserHandler {
	return &UserHandler{
		Store:      store,
		Config:     config,
		TokenMaker: tokenMaker,
	}
}

var validate = validator.New()

func NewUserResponse(user db.User) userResponse {
	return userResponse{
		ID:        user.ID,
		FullName:  user.FullName,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
	}
}

type LoginUserRequest struct {
	PhoneNumber string `json:"phone" validate:"required"`
	Password    string `json:"password" validate:"required"`
}

type LoginUserResponse struct {
	SessionID uuid.UUID    `json:"session_id"`
	User      userResponse `json:"user"`
}

func (u *UserHandler) RegisterUser(c fiber.Ctx) error {
	var req CreateUserRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	arg := db.CreateUserParams{
		FullName: req.FullName,
		Phone:    req.PhoneNumber,
		Password: hashedPassword,
	}

	user, err := u.Store.CreateUser(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	rsp := NewUserResponse(user)

	return c.Status(fiber.StatusCreated).JSON(rsp)
}

func (u *UserHandler) LoginUser(c fiber.Ctx) error {
	var req LoginUserRequest

	// Bind request body
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate request
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Find user
	user, err := u.Store.GetUserByPhone(c.Context(), req.PhoneNumber)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid phone or password",
		})
	}

	// Check password
	if err := util.CheckPassword(req.Password, user.Password); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid phone or password",
		})
	}

	// Create access token
	accessToken, accessPayload, err := u.TokenMaker.CreateToken(
		req.PhoneNumber,
		user.ID,
		u.Config.ACCESS_TOKEN_DURATION,
	)

	refreshToken, refreshPayload, err := u.TokenMaker.CreateToken(
		req.PhoneNumber,
		user.ID,
		u.Config.REFRESH_TOKEN_DURATION,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create session
	session, err := u.Store.CreateSession(c.Context(), db.CreateSessionParams{
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

		ExpiresAt: pgtype.Timestamp{
			Time:  refreshPayload.ExpiredAt,
			Valid: true,
		},
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// -----------------------------
	// Access Token Cookie
	// -----------------------------
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  accessPayload.ExpiredAt,
		HTTPOnly: true,
		Secure:   false, // localhost -> true in production
		SameSite: "Lax",
		Path:     "/",
	})

	// -----------------------------
	// Refresh Token Cookie
	// -----------------------------
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  refreshPayload.ExpiredAt,
		HTTPOnly: true,
		Secure:   false, // localhost -> true in production
		SameSite: "Lax",
		Path:     "/",
	})

	// Response
	rsp := LoginUserResponse{
		SessionID: session.ID,
		User:      NewUserResponse(user),
	}

	return c.Status(fiber.StatusOK).JSON(rsp)
}

// internal/handlers/user_handler.go — اضافه کن به فایل فعلی

// GET /users/:id
func (u *UserHandler) GetUser(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	payload := c.Locals("payload").(*token.Payload)

	user, err := u.Store.GetUserByPhone(c.Context(), payload.Phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	if user.ID != id {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied"})
	}

	return c.JSON(NewUserResponse(user))
}

// GET /users
func (u *UserHandler) GetAllUsers(c fiber.Ctx) error {
	users, err := u.Store.GetAllUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	rsp := make([]userResponse, len(users))
	for i, user := range users {
		rsp[i] = NewUserResponse(user)
	}

	return c.JSON(rsp)
}

func (u *UserHandler) GetMe(c fiber.Ctx) error {
	payload := c.Locals("payload").(*token.Payload)

	user, err := u.Store.GetUserByPhone(c.Context(), payload.Phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	return c.JSON(NewUserResponse(user))
}
