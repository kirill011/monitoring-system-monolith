package messages

import (
	"fmt"
	"monolith/internal/models"
	"monolith/internal/services"

	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gofiber/fiber/v3"
)

const (
	localID = "localID"
)

type messagesHandler struct {
	natsHandlers services.Messages
	metrics      prometheus.Counter
	ingrMetrics  prometheus.Counter
}

type Config struct {
	NatsHandlers services.Messages
	Metrics      prometheus.Counter
}

func NewMessagesHandler(cfg *Config) *messagesHandler {
	ingressRequests := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ingress_requests_total",
			Help: "Total incoming requests",
		},
	)

	prometheus.MustRegister(ingressRequests)
	return &messagesHandler{
		natsHandlers: cfg.NatsHandlers,
		metrics:      cfg.Metrics,
		ingrMetrics:  ingressRequests,
	}
}

func (h *messagesHandler) InitMessagesRoutes(api fiber.Router) {
	servicesRoute := api.Group("/messages")
	servicesRoute.Post("/send_msg", h.sendMsg)
}

type (
	sendMsgReq struct {
		Message     string `form:"message" 		json:"message" 		validate:"required" 	xml:"message"`
		MessageType string `form:"message_type" json:"message_type" validate:"required" 	xml:"message_type"`
		Component   string `form:"component" 	json:"component" 	validate:"required" 	xml:"component"`
		Address     string `form:"address" 		json:"address" 		validate:"required,ip" 	xml:"address"`
	}
)

func (h *messagesHandler) sendMsg(ctx fiber.Ctx) error {
	h.ingrMetrics.Inc()
	body := sendMsgReq{
		Message:     "",
		MessageType: "",
		Component:   "",
		Address:     "",
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			fmt.Errorf("ctx.Bind().Body: %w", err).Error(),
		)
	}

	_, _, err := h.natsHandlers.Create(models.Message{
		Message:     body.Message,
		MessageType: body.MessageType,
		Component:   body.Component,
		DeviceIP:    body.Address,
	})
	if err != nil {
		return fiber.NewError(
			fiber.StatusInternalServerError,
			fmt.Errorf("h.nats.PublishGetAllByDeviceId: %w", err).Error(),
		)
	}

	jsonResponse, err := jsoniter.Marshal(
		fiber.StatusAccepted,
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
	h.metrics.Inc()

	return nil
}
