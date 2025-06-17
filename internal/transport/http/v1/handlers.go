package v1

import (
	"monolith/internal/services"
	authHandlers "monolith/internal/transport/http/v1/auth"
	devicesHandlers "monolith/internal/transport/http/v1/devices"
	reportsHandlers "monolith/internal/transport/http/v1/reports"
	tagsHandlers "monolith/internal/transport/http/v1/tags"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

type Handler struct {
	jwtKey          string
	tokenLifeTime   time.Duration
	authService     services.Auth
	devicesService  services.Devices
	tagsHandlers    services.Tags
	reportsHandlers services.Messages
	deviceChecker   services.DevicesHandler

	metrics prometheus.Counter
}

type Config struct {
	JwtKey          string
	TokenLifeTime   time.Duration
	AuthHandlers    services.Auth
	DevicesHandlers services.Devices
	TagsHandlers    services.Tags
	ReportsHandlers services.Messages
	DeviceChecker   services.DevicesHandler

	Metrics prometheus.Counter
}

func NewHandler(cfg Config) *Handler {
	return &Handler{
		jwtKey:          cfg.JwtKey,
		tokenLifeTime:   cfg.TokenLifeTime,
		authService:     cfg.AuthHandlers,
		devicesService:  cfg.DevicesHandlers,
		tagsHandlers:    cfg.TagsHandlers,
		reportsHandlers: cfg.ReportsHandlers,
		deviceChecker:   cfg.DeviceChecker,
	}
}

func (h *Handler) InitRouter(routeV1 fiber.Router) {
	authHandlers.NewAuthHandler(&authHandlers.Config{
		AuthService:   h.authService,
		JWTKey:        h.jwtKey,
		TokenLifeTime: h.tokenLifeTime,
	}).InitAuthRoutes(routeV1)

	devicesHandlers.NewDevicesHandler(&devicesHandlers.Config{
		DevicesService: h.devicesService,
		Messages:       h.reportsHandlers,
		DeviceChecker:  h.deviceChecker,
		JWTKey:         h.jwtKey,
	}).InitDevicesRoutes(routeV1)

	tagsHandlers.NewTagsHandler(&tagsHandlers.Config{
		NatsHandlers: h.tagsHandlers,
		Messages:     h.reportsHandlers,
		JWTKey:       h.jwtKey,
	}).InitTagsRoutes(routeV1)

	reportsHandlers.NewReportsHandler(&reportsHandlers.Config{
		NatsHandlers: h.reportsHandlers,
		JWTKey:       h.jwtKey,
	}).InitReportsRoutes(routeV1)

	devicesHandlers.NewDevicesHandler(&devicesHandlers.Config{
		DevicesService: h.devicesService,
		Messages:       h.reportsHandlers,
		JWTKey:         h.jwtKey,
	}).InitDevicesRoutes(routeV1)
}
