package devicechecker

import (
	"fmt"
	"monolith/internal/models"
	"monolith/internal/services"
	"net/http"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type devicechecekrHandler struct {
	deviceService  services.DeviceHandler
	messageService services.Messages

	deviceCheckPeriod int
	cron              *cron.Cron

	log *zap.Logger
}

type Config struct {
	DeviceService  services.DeviceHandler
	MessageService services.Messages

	DeviceCheckPeriod int
	Logger            *zap.Logger
}

func NewDeviceCheckerHandler(cfg *Config) *devicechecekrHandler {
	cron := cron.New(cron.WithSeconds())
	return &devicechecekrHandler{
		deviceService:     cfg.DeviceService,
		deviceCheckPeriod: cfg.DeviceCheckPeriod,
		cron:              cron,
		log:               cfg.Logger,
	}
}

func (dch *devicechecekrHandler) Start() {
	_, err := dch.cron.AddFunc(fmt.Sprintf("*/%d * * * * *", dch.deviceCheckPeriod),
		func() {
			ips := dch.deviceService.GetDevicesIPs()
			for _, ip := range ips {
				id, ok := dch.deviceService.GetDeviceIDByIp(ip)
				if !ok {
					dch.log.Warn("device not found", zap.String("ip", ip))
					continue
				}
				resp, err := http.Get(fmt.Sprintf("http://%s/healthcheck", ip))
				if err != nil {
					dch.messageService.Create(models.Message{
						DeviceId:    id,
						Message:     fmt.Sprintf("unable to connect to device %s", ip),
						MessageType: "error",
						Component:   "General",
					})
					return
				}

				if resp.StatusCode != http.StatusOK {
					dch.messageService.Create(models.Message{
						DeviceId:    id,
						Message:     fmt.Sprintf("device %s status is not OK", ip),
						MessageType: "error",
						Component:   "General",
					})
				}
			}
		},
	)
	if err != nil {
		dch.log.Error("error adding device checker to cron", zap.Error(err))
	}

	dch.cron.Start()
}
