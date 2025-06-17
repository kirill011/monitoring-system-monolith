package devices

import (
	"context"
	"errors"
	"fmt"
	"monolith/internal/models"
	"monolith/internal/services"

	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	localID = "localID"
)

type devicesHandler struct {
	devicesService services.Devices
	messages       services.Messages
	deviceChecker  services.DevicesHandler
	jwtKey         string
}

type Config struct {
	JWTKey         string
	DevicesService services.Devices
	Messages       services.Messages
	DeviceChecker  services.DevicesHandler
}

func NewDevicesHandler(cfg *Config) *devicesHandler {
	return &devicesHandler{
		jwtKey:         cfg.JWTKey,
		devicesService: cfg.DevicesService,
		messages:       cfg.Messages,
		deviceChecker:  cfg.DeviceChecker,
	}
}

func (h *devicesHandler) InitDevicesRoutes(api fiber.Router) {
	servicesRoute := api.Group("/devices", h.deserializeMW)
	servicesRoute.Post("/create", h.create)
	servicesRoute.Get("/read", h.read)
	servicesRoute.Put("/update", h.update)
	servicesRoute.Delete("/delete", h.delete)

}

type (
	registerReq struct {
		Name        string  `form:"name"         json:"name"         validate:"required"       xml:"name"`
		DeviceType  string  `form:"device_type"  json:"device_type"  validate:"required"       xml:"device_type"`
		Address     string  `form:"address"      json:"address"      validate:"required,ip"    xml:"address"`
		Responsible []int32 `form:"responsible"  json:"responsible"  validate:"required"       xml:"responsible"`
	}

	registerResp struct {
		Data models.Device `json:"data"`
	}
)

func (h *devicesHandler) create(ctx fiber.Ctx) error {
	body := registerReq{
		Name:        "",
		DeviceType:  "",
		Address:     "",
		Responsible: []int32{},
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	res, err := h.devicesService.Create(
		context.Background(),
		models.Device{
			Name:        body.Name,
			DeviceType:  body.DeviceType,
			Address:     body.Address,
			Responsible: body.Responsible,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishCreate: %w", err).Error(),
		)
	}

	h.messages.UpdateDevices()
	h.deviceChecker.UpdateDevices()

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
	read struct {
		Data services.ReadDevicesResult `json:"data"`
	}
)

func (h *devicesHandler) read(ctx fiber.Ctx) error {
	idLocals := ctx.Locals(localID)
	_, ok := idLocals.(int) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("idLocals.(int): invalid token").Error(),
		)
	}

	res, err := h.devicesService.Read(context.Background())
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.devicesService.PublishRead: %w", err).Error(),
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
		ID          int32   `form:"id"           json:"id"           validate:"required"       xml:"id"`
		Name        string  `form:"name"         json:"name"         validate:"omitempty"       xml:"name"`
		DeviceType  string  `form:"device_type"  json:"device_type"  validate:"omitempty"       xml:"device_type"`
		Address     string  `form:"address"      json:"address"      validate:"omitempty,ip"    xml:"address"`
		Responsible []int32 `form:"responsible"  json:"responsible"  validate:"omitempty"       xml:"responsible"`
	}

	updateResp struct {
		Data int `json:"data"`
	}
)

func (h *devicesHandler) update(ctx fiber.Ctx) error {
	idLocals := ctx.Locals(localID)
	_, ok := idLocals.(int) //nolint:varnamelen
	if !ok {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			errors.New("idLocals.(int): invalid token").Error(),
		)
	}

	body := updateReq{
		Name:        "",
		DeviceType:  "",
		Address:     "",
		Responsible: []int32{},
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	if body.DeviceType == "" && body.Address == "" &&
		len(body.Responsible) == 0 && body.Name == "" {
		return fiber.NewError(
			fiber.StatusBadRequest,
			errors.New(`body.DeviceType == "" && body.Address == "" && len(body.Responsible) == 0 && body.Name == ""`).Error(),
		)
	}

	err := h.devicesService.Update(
		context.Background(),
		services.UpdateDeviceParams{
			ID:          body.ID,
			Name:        &body.Name,
			DeviceType:  &body.DeviceType,
			Address:     &body.Address,
			Responsible: body.Responsible,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.devicesService.Update: %w", err).Error(),
		)
	}
	h.messages.UpdateDevices()
	h.deviceChecker.UpdateDevices()

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

func (h *devicesHandler) delete(ctx fiber.Ctx) error {
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

	err := h.devicesService.Delete(
		context.Background(),
		int32(body.ID),
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.devicesService.PublishDelete: %w", err).Error(),
		)
	}

	h.messages.UpdateDevices()
	h.deviceChecker.UpdateDevices()

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

func (h *devicesHandler) deserializeMW(ctx fiber.Ctx) error {
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
