package services

import (
	"context"
	"monolith/internal/models"
	"monolith/internal/repo"
	"sync"
)

type DevicesHandler interface {
	UpdateDevices()
	GetDeviceIDByIp(address string) (int32, bool)
	GetDevicesIPs() []string
}

type DeviceHandler struct {
	repo repo.Devices
}

func NewDeviceHandler(devices repo.Devices) *DeviceHandler {
	deviceHandler := DeviceHandler{
		repo: devices,
	}

	deviceHandler.UpdateDevices()
	return &deviceHandler
}

var (
	devicesMutex sync.Mutex
	devicesByIp  = map[string]models.Device{}
)

func (ds *DeviceHandler) UpdateDevices() {
	tx, err := ds.repo.BeginTx(context.Background())
	if err != nil {
		return
	}
	defer tx.Rollback()

	devices, err := tx.Read(context.Background())
	if err != nil {
		return
	}
	if err := tx.Commit(); err != nil {
		return
	}

	devicesMutex.Lock()
	defer devicesMutex.Unlock()

	devicesByIp = make(map[string]models.Device, len(devices.Devices))
	for _, d := range devices.Devices {
		devicesByIp[d.Address] = d
	}
}

func (ds *DeviceHandler) GetDeviceIDByIp(address string) (int32, bool) {
	device, ok := devicesByIp[address]
	return device.ID, ok
}

func (ds *DeviceHandler) GetDevicesIPs() []string {
	res := make([]string, 0, len(devicesByIp))
	for ip := range devicesByIp {
		res = append(res, ip)
	}

	return res
}
