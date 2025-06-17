package reports

import (
	"errors"
	"fmt"
	"monolith/internal/models"
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

type reportsHandler struct {
	natsHandlers services.Messages
	jwtKey       string
}

type Config struct {
	JWTKey       string
	NatsHandlers services.Messages
}

func NewReportsHandler(cfg *Config) *reportsHandler {
	return &reportsHandler{
		jwtKey:       cfg.JWTKey,
		natsHandlers: cfg.NatsHandlers,
	}
}

func (h *reportsHandler) InitReportsRoutes(api fiber.Router) {
	servicesRoute := api.Group("/reports", h.deserializeMW)
	servicesRoute.Get("/get_all_by_device_id", h.getAllByDeviceId)
	servicesRoute.Get("/get_all_by_period", h.getAllByPeriod)
	servicesRoute.Get("/get_count_by_message_type", h.getCountByMessageType)
	servicesRoute.Get("/month_report", h.getMonthReport)

}

type (
	getAllByDeviceIdReq struct {
		DeviceID int `form:"device_id" json:"device_id" validate:"required" xml:"device_id"`
	}

	getAllByDeviceIdResp struct {
		Data []services.ReportGetAllByDeviceId `json:"data"`
	}
)

func (h *reportsHandler) getAllByDeviceId(ctx fiber.Ctx) error {
	body := getAllByDeviceIdReq{
		DeviceID: 0,
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	res, err := h.natsHandlers.GetAllByDeviceId(
		int32(body.DeviceID),
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishGetAllByDeviceId: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&getAllByDeviceIdResp{
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
	getAllByPeriodReq struct {
		StartTime time.Time `form:"start_time" json:"start_time" validate:"required" xml:"start_time"`
		EndTime   time.Time `form:"end_time"   json:"end_time"   validate:"required" xml:"end_time"`
	}

	getAllByPeriodResp struct {
		Data []services.ReportGetAllByPeriod `json:"data"`
	}
)

func (h *reportsHandler) getAllByPeriod(ctx fiber.Ctx) error {
	body := getAllByPeriodReq{
		StartTime: time.Time{},
		EndTime:   time.Time{},
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	res, err := h.natsHandlers.GetAllByPeriod(
		services.MessagesGetAllByPeriodOpts{
			StartTime: body.StartTime,
			EndTime:   body.EndTime,
		},
	)
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishGetAllByPeriod: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&getAllByPeriodResp{
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
	getCountByMessageTypeReq struct {
		MessageType string `form:"message_type" json:"message_type" validate:"required" xml:"message_type"`
	}

	getCountByMessageTypeResp struct {
		Data []services.ReportGetCountByMessageType `json:"data"`
	}
)

func (h *reportsHandler) getCountByMessageType(ctx fiber.Ctx) error {
	body := getCountByMessageTypeReq{
		MessageType: "",
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	res, err := h.natsHandlers.GetCountByMessageType(
		body.MessageType,
	)

	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishGetCountByMessageType: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&getCountByMessageTypeResp{
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
	getMonthReportResp struct {
		Data []models.MonthReportRow `json:"data"`
	}
)

func (h *reportsHandler) getMonthReport(ctx fiber.Ctx) error {
	res, err := h.natsHandlers.MonthReport()
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishGetMonthReport: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		&getMonthReportResp{
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

func (h *reportsHandler) deserializeMW(ctx fiber.Ctx) error {
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
