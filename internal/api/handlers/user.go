package handlers

import (
	"net/netip"
	"time"

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
	FirstName   string `json:"first_name" validate:"required"`
	LastName    string `json:"last_name" validate:"required"`
	PhoneNumber string `json:"phone" validate:"required"`
	Password    string `json:"password" validate:"required"`
}
type userResponse struct {
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
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
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
	}
}

type LoginUserRequest struct {
	PhoneNumber string `json:"phone" validate:"required"`
	Password    string `json:"password" validate:"required"`
}

type LoginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (u *UserHandler) RegisterUser(c fiber.Ctx) error {
	var req LoginUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	arg := db.CreateUserParams{
		Phone:    req.PhoneNumber,
		Password: hashedPassword,
	}
	user, err := u.Store.CreateUser(c.Context(), arg)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	rsp := NewUserResponse(user)
	return c.Status(fiber.StatusCreated).JSON(rsp)
}

func (u *UserHandler) LoginUser(c fiber.Ctx) error {
	var req LoginUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	user, err := u.Store.GetUserByPhone(c.Context(), req.PhoneNumber)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	err = util.CheckPassword(req.Password, user.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}

	accessToken, accessPayload, err := u.TokenMaker.CreateToken(req.PhoneNumber, u.Config.ACCESS_TOKEN_DURATION)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	refreshToken, refreshPayload, err := u.TokenMaker.CreateToken(req.PhoneNumber, u.Config.REFRESH_TOKEN_DURATION)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}

	session, err := u.Store.CreateSession(c.Context(), db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Phone:        user.Phone,
		RefreshToken: refreshToken,
		UserAgent:    pgtype.Text{String: string(c.Request().Header.UserAgent()), Valid: true},
		ClientIp: func() *netip.Addr {
			addr, err := netip.ParseAddr(c.IP())
			if err != nil {
				return nil
			}
			return &addr
		}(),
		IsBlocked: false,
		ExpiresAt: pgtype.Timestamp{Time: refreshPayload.ExpiredAt, Valid: true}})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})

	}
	rsp := LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  NewUserResponse(user),
	}
	return c.Status(fiber.StatusBadRequest).JSON(rsp)

}
