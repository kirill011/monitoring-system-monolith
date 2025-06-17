package auth

import (
	"context"
	"errors"
	"fmt"
	"monolith/internal/services"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	localID = "localID"
)

type authHandler struct {
	authService   services.Auth
	jwtKey        string
	tokenLifeTime time.Duration
}

type Config struct {
	JWTKey        string
	AuthService   services.Auth
	TokenLifeTime time.Duration
}

func NewAuthHandler(cfg *Config) *authHandler {
	return &authHandler{
		jwtKey:        cfg.JWTKey,
		authService:   cfg.AuthService,
		tokenLifeTime: cfg.TokenLifeTime,
	}
}

func (h *authHandler) InitAuthRoutes(api fiber.Router) {
	servicesRoute := api.Group("/auth")
	servicesRoute.Post("/register", h.register)
	servicesRoute.Get("/sign_in", h.authorize)

	servicesRoute = api.Group("/users", h.deserializeMW)
	servicesRoute.Get("/read", h.read)
	servicesRoute.Put("/update", h.update)
	servicesRoute.Delete("/delete", h.delete)

}

type (
	registerReq struct {
		Name     string `form:"name"         json:"name"         validate:"required"       xml:"name"`
		Email    string `form:"email"        json:"email"        validate:"required,email" xml:"email"`
		Password string `form:"password"     json:"password"     validate:"required"       xml:"password"`
	}

	registerResp struct {
		Data services.CreateUserResult `json:"data"`
	}
)

func (h *authHandler) register(ctx fiber.Ctx) error {
	body := registerReq{
		Name:     "",
		Email:    "",
		Password: "",
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	res, err := h.authService.Create(
		context.Background(),
		services.CreateUserParams{
			Email:    body.Email,
			Password: body.Password,
			Name:     body.Name,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishAuthRegister: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&registerResp{
			Data: res,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("json.Marshal: %w", err).Error(),
		)
	}

	if err = ctx.Status(fiber.StatusOK).Send(jsonResponse); err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("ctx.Send: %w", err).Error(),
		)
	}

	return nil
}

type (
	authorizeReq struct {
		Email    string `form:"email"    json:"email"    validate:"required,email" xml:"email"`
		Password string `form:"password" json:"password" validate:"required"       xml:"password"`
	}

	authorizeResp struct {
		JWT string `json:"jwt"`
	}
)

func (h *authHandler) authorize(ctx fiber.Ctx) error {
	body := authorizeReq{
		Email:    "",
		Password: "",
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	userID, err := h.authService.Authorize(
		context.Background(),
		services.AuthorizeParams{
			Email:    body.Email,
			Password: body.Password,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			fmt.Errorf("h.natsHandlers.PublishAuth: %w", err).Error(),
		)
	}

	jwt := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		localID: userID,
		"exp":   time.Now().Add(h.tokenLifeTime).Unix(),
	})

	token, err := jwt.SignedString([]byte(h.jwtKey))
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("jwt.SignedString: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(&authorizeResp{
		JWT: token,
	})
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("json.Marshal: %w", err).Error(),
		)
	}

	if err = ctx.Status(fiber.StatusOK).Send(jsonResponse); err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("ctx.Send: %w", err).Error(),
		)
	}

	return nil
}

type (
	read struct {
		Data services.ReadResult `json:"data"`
	}
)

func (h *authHandler) read(ctx fiber.Ctx) error {
	idLocals := ctx.Locals(localID)
	_, ok := idLocals.(int) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("idLocals.(int): invalid token").Error(),
		)
	}

	res, err := h.authService.Read(context.Background())
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.natsHandlers.PublishAuthRead: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&read{
			Data: res,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("json.Marshal: %w", err).Error(),
		)
	}

	if err = ctx.Status(fiber.StatusOK).Send(jsonResponse); err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("ctx.Send: %w", err).Error(),
		)
	}
	return nil
}

type (
	updateReq struct {
		ID       int32  `form:"id"       json:"id"       validate:"required"             xml:"id"`
		Name     string `form:"name"     json:"name"     validate:"omitempty"            xml:"name"`
		Email    string `form:"email"    json:"email"    validate:"omitempty,email"      xml:"email"`
		Password string `form:"password" json:"password" validate:"omitempty"            xml:"password"`
	}

	updateResp struct {
		Data int `json:"data"`
	}
)

func (h *authHandler) update(ctx fiber.Ctx) error {
	idLocals := ctx.Locals(localID)
	_, ok := idLocals.(int) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("idLocals.(int): invalid token").Error(),
		)
	}

	body := updateReq{
		Name:     "",
		Email:    "",
		Password: "",
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	if body.Name == "" && body.Email == "" && body.Password == "" {
		return fiber.NewError(
			fiber.StatusBadRequest,
			errors.New("updOpt.Name == nil").Error(),
		)
	}

	err := h.authService.Update(
		context.Background(),
		services.UpdateUsersParams{
			ID:       body.ID,
			Name:     &body.Name,
			Email:    &body.Email,
			Password: &body.Password,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.natsHandlers.PublishAuthUpdate: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&updateResp{
			Data: fiber.StatusOK,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("json.Marshal: %w", err).Error(),
		)
	}

	if err = ctx.Status(fiber.StatusOK).Send(jsonResponse); err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("ctx.Send: %w", err).Error(),
		)
	}
	return nil
}

type (
	deleteReq struct {
		ID int `form:"id"     json:"id"     validate:"required"            xml:"id"`
	}
	deleteResp struct {
		Data int `json:"data"`
	}
)

func (h *authHandler) delete(ctx fiber.Ctx) error {
	idLocals := ctx.Locals(localID)
	_, ok := idLocals.(int) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("idLocals.(int): invalid token").Error(),
		)
	}

	body := deleteReq{
		ID: 0,
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	err := h.authService.Delete(
		context.Background(),
		int32(body.ID),
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.natsHandlers.PublishAuthDelete: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&deleteResp{
			Data: fiber.StatusOK,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("json.Marshal: %w", err).Error(),
		)
	}

	if err = ctx.Status(fiber.StatusOK).Send(jsonResponse); err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("ctx.Send: %w", err).Error(),
		)
	}

	return nil
}

func (h *authHandler) deserializeMW(ctx fiber.Ctx) error {
	tokenString := ctx.Get("Authorization")

	if tokenString == "" {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("tokenString is empty").Error(),
		)
	}

	tokenString = strings.ReplaceAll(tokenString, "Bearer ", "")
	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return []byte(h.jwtKey), nil
	})
	if err != nil {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			fmt.Errorf("jwt.Parse: %w", err).Error(),
		)
	}

	claims, ok := token.Claims.(jwt.MapClaims) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("token.Claims.(jwt.MapClaims): invalid token").Error(),
		)
	}

	userID, ok := claims[localID].(float64) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("claims["+localID+"].(float64): invalid token").Error(),
		)
	}

	ctx.Locals(localID, int(userID))

	return ctx.Next() //nolint:wrapcheck
}
