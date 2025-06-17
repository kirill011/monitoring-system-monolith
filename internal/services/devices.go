package services

import (
	"context" //nolint:gosec
	"fmt"

	"monolith/internal/models"
	"monolith/internal/repo"
)

type Devices interface {
	Create(ctx context.Context, params models.Device) (models.Device, error)
	Read(ctx context.Context) (ReadDevicesResult, error)
	Update(ctx context.Context, params UpdateDeviceParams) error
	Delete(ctx context.Context, deviceID int32) error
	GetResponsible(ctx context.Context, deviceID int32) ([]int32, error)
}

type DeviceService struct {
	repo repo.Devices
}

func NewDevicesService(r repo.Devices) Devices {
	return &DeviceService{
		repo: r,
	}
}

func (s *DeviceService) Create(ctx context.Context, params models.Device) (models.Device, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return models.Device{}, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.Create(params)
	if err != nil {
		return models.Device{}, fmt.Errorf("tx.Create: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return models.Device{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return ret, nil
}

type (
	ReadDevicesResult struct {
		Devices []models.Device
	}
)

func (s *DeviceService) Read(ctx context.Context) (ReadDevicesResult, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return ReadDevicesResult{}, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.Read(ctx)
	if err != nil {
		return ReadDevicesResult{}, fmt.Errorf("tx.Read: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return ReadDevicesResult{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return ReadDevicesResult{
		Devices: ret.Devices,
	}, nil
}

type (
	UpdateDeviceParams struct {
		ID          int32
		Name        *string
		DeviceType  *string
		Address     *string
		Responsible []int32
	}
)

func (s *DeviceService) Update(ctx context.Context, params UpdateDeviceParams) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	err = tx.Update(ctx, repo.UpdateDeviceOpts{
		ID:          params.ID,
		Name:        params.Name,
		DeviceType:  params.DeviceType,
		Address:     params.Address,
		Responsible: params.Responsible,
	})
	if err != nil {
		return fmt.Errorf("s.repo.Update: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func (s *DeviceService) Delete(ctx context.Context, deviceID int32) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	err = tx.Delete(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("tx.Delete: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func (s *DeviceService) GetResponsible(ctx context.Context, deviceID int32) ([]int32, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.GetResponsible(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("tx.GetResponsible: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return ret, nil
}
