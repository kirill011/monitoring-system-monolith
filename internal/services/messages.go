package services

import (
	"context"
	"fmt"
	"monolith/internal/models"
	"monolith/internal/repo"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type Messages interface {
	UpdateDevices()
	UpdateTags()
	Create(opts models.Message) (CreateMessageResponse, bool, error)
	GetAllByPeriod(opts MessagesGetAllByPeriodOpts) ([]ReportGetAllByPeriod, error)
	GetAllByDeviceId(deviceID int32) ([]ReportGetAllByDeviceId, error)
	GetCountByMessageType(messageType string) ([]ReportGetCountByMessageType, error)
	MonthReport() ([]models.MonthReportRow, error)
}
type MessagesService struct {
	messageRepo        repo.Messages
	tagRepo            repo.Tags
	devicesRepo        repo.Devices
	cron               *cron.Cron
	notificationPeriod time.Duration

	log *zap.Logger
}

type MessagesServiceConfig struct {
	MessageRepo repo.Messages
	TagRepo     repo.Tags
	DevicesRepo repo.Devices

	NotificationPeriod time.Duration
	Log                *zap.Logger
}

func NewMessagesService(cfg MessagesServiceConfig) Messages {
	messagesService := &MessagesService{
		messageRepo: cfg.MessageRepo,
		tagRepo:     cfg.TagRepo,
		devicesRepo: cfg.DevicesRepo,
		log:         cfg.Log,
	}
	messagesService.UpdateTags()
	return messagesService
}

var (
	deviceByDeviceIDMutex sync.Mutex
	deviceByDeviceID      = map[int32]models.Device{}
)

var defaultDevice = models.Device{
	ID:          -1,
	Name:        "unknown device",
	DeviceType:  "unknown device",
	Address:     "unknown device",
	Responsible: []int32{},
}

func (s *MessagesService) SetDevices(newDevices []models.Device) {
	deviceByDeviceIDMutex.Lock()
	defer deviceByDeviceIDMutex.Unlock()
	deviceByDeviceID = make(map[int32]models.Device)
	for _, device := range newDevices {
		deviceByDeviceID[device.ID] = device
	}

	s.log.Debug("set devices", zap.Any("devices", deviceByDeviceID))
}

func (s *MessagesService) UpdateDevices() {
	tx, err := s.devicesRepo.BeginTx(context.Background())
	if err != nil {
		s.log.Error("tx.BeginTx", zap.Error(err))
		return
	}
	defer tx.Rollback()

	devices, err := tx.Read(context.Background())

	err = tx.Commit()
	if err != nil {
		s.log.Error("tx.Commit", zap.Error(err))
	}

	s.SetDevices(devices.Devices)
}

var (
	tagsMutex sync.Mutex
	tags      = map[int32][]models.Tag{}
)

func (ms *MessagesService) UpdateTags() {
	tx, err := ms.tagRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return
	}
	defer tx.Rollback()

	tagsMutex.Lock()
	defer tagsMutex.Unlock()

	tags = make(map[int32][]models.Tag)
	dbTags, err := tx.Read(context.Background())
	if err != nil {
		ms.log.Error("ms.tagRepo.Read", zap.Error(err))
		return
	}

	for _, dbTag := range dbTags.Tags {
		switch dbTag.CompareType {
		case "=":
			dbTag.CompareFunc = func(a string, b string) bool { return a == b }
		case "<":
			dbTag.CompareFunc = func(a string, b string) bool {
				first, err := strconv.ParseFloat(a, 64)
				if err != nil {
					ms.log.Error("strconv.ParseFloat", zap.Error(err))
				}

				second, err := strconv.ParseFloat(b, 64)
				if err != nil {
					ms.log.Error("strconv.ParseFloat", zap.Error(err))
				}
				return first < second
			}

		case ">":
			dbTag.CompareFunc = func(a string, b string) bool {
				first, err := strconv.ParseFloat(a, 64)
				if err != nil {
					ms.log.Error("strconv.ParseFloat", zap.Error(err))
				}

				second, err := strconv.ParseFloat(b, 64)
				if err != nil {
					ms.log.Error("strconv.ParseFloat", zap.Error(err))
				}
				return first > second
			}
		default:
			ms.log.Error("unknown compare type", zap.String("compare type", dbTag.CompareType))
			continue
		}

		dbTag.CompiledRegexp, err = regexp.Compile(dbTag.Regexp)
		if err != nil {
			ms.log.Error("regexp.Compile", zap.Error(err))
			continue
		}

		if tags[dbTag.DeviceId] == nil {
			tags[dbTag.DeviceId] = make([]models.Tag, 0)
		}
		tags[dbTag.DeviceId] = append(tags[dbTag.DeviceId], dbTag)
	}
}

type (
	CreateMessageResponse struct {
		Text    string
		Subject string
	}
)

func (ms *MessagesService) Create(opts models.Message) (CreateMessageResponse, bool, error) {
	resp, err := ms.handleMessage(opts)
	if err != nil {
		return CreateMessageResponse{}, false, fmt.Errorf("ms.handleMessage: %w", err)
	}

	tx, err := ms.messageRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return CreateMessageResponse{}, false, fmt.Errorf("ms.messageRepo.Create: %w", err)
	}
	defer tx.Rollback()

	if err := tx.Create(resp.Message); err != nil {
		return CreateMessageResponse{}, false, fmt.Errorf("tx.Create: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return CreateMessageResponse{}, false, fmt.Errorf("tx.Commit: %w", err)
	}

	return CreateMessageResponse{
		Subject: resp.Subject,
		Text:    resp.Text,
	}, resp.NeedNotify, nil
}

type handleMessageResponse struct {
	Text       string
	Subject    string
	NeedNotify bool
	Message    models.Message
}

func (ms *MessagesService) handleMessage(message models.Message) (handleMessageResponse, error) {
	for _, tag := range tags[message.DeviceId] {
		finded := tag.CompiledRegexp.FindStringSubmatch(message.Message)
		if len(finded) == 0 {
			continue
		}
		if tag.Compare(finded[tag.ArrayIndex]) {
			if tag.CompareType == ">" || tag.CompareType == "<" {
				ms.handleReversedTag(tag)
			}
			message.SeverityLevel = tag.SeverityLevel
		}
		return handleMessageResponse{
			Text:       message.Message,
			Subject:    tag.Subject,
			NeedNotify: true,
			Message:    message,
		}, nil
	}

	return handleMessageResponse{Message: message, NeedNotify: false}, nil
}

func (ms *MessagesService) handleReversedTag(tag models.Tag) {
	if tag.Subject == "OK" {
		err := ms.tagRepo.Delete(context.Background(), tag.ID)
		if err != nil {
			ms.log.Error("ms.tagRepo.Delete", zap.Error(err), zap.Int32("tag id", tag.ID))
		}
	} else {
		reversedTag := tag
		reversedTag.Subject = "OK"
		reversedTag.SeverityLevel = "info"
		switch tag.CompareType {
		case ">":
			reversedTag.CompareType = "<"
		case "<":
			reversedTag.CompareType = ">"
		}
		_, err := ms.tagRepo.Create(reversedTag)
		if err != nil {
			ms.log.Error("ms.tagRepo.Create", zap.Error(err), zap.Any("tag", reversedTag))
		}
	}
}

type (
	MessagesGetAllByPeriodOpts struct {
		StartTime time.Time
		EndTime   time.Time
	}
	ReportGetAllByPeriod struct {
		DeviceID    int32
		Name        string
		DeviceType  string
		Address     string
		Responsible []int32
		GotAt       time.Time
		Message     string
		MessageType string
	}
)

func (ms *MessagesService) GetAllByPeriod(opts MessagesGetAllByPeriodOpts) ([]ReportGetAllByPeriod, error) {
	tx, err := ms.messageRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return nil, fmt.Errorf("ms.messageRepo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.GetAllByPeriod(
		repo.MessagesGetAllByPeriodOpts{
			StartTime: opts.StartTime,
			EndTime:   opts.EndTime,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("tx.GetAllByPeriod: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return lo.Map(result, func(r models.Message, indx int) ReportGetAllByPeriod {
		if dev, ok := deviceByDeviceID[r.DeviceId]; ok {
			return ReportGetAllByPeriod{
				DeviceID:    r.DeviceId,
				Name:        dev.Name,
				DeviceType:  dev.DeviceType,
				Address:     dev.Address,
				Responsible: dev.Responsible,
				Message:     r.Message,
				MessageType: r.MessageType,
				GotAt:       r.GotAt,
			}
		}
		return ReportGetAllByPeriod{
			DeviceID:    r.DeviceId,
			Name:        defaultDevice.Name,
			DeviceType:  defaultDevice.DeviceType,
			Address:     defaultDevice.Address,
			Responsible: defaultDevice.Responsible,
			Message:     r.Message,
			MessageType: r.MessageType,
			GotAt:       r.GotAt,
		}
	}), nil
}

type (
	ReportGetAllByDeviceId struct {
		DeviceID    int32
		Name        string
		DeviceType  string
		Address     string
		Responsible []int32
		GotAt       time.Time
		Message     string
		MessageType string
	}
)

func (ms *MessagesService) GetAllByDeviceId(deviceID int32) ([]ReportGetAllByDeviceId, error) {
	tx, err := ms.messageRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return nil, fmt.Errorf("ms.messageRepo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.GetAllByDeviceId(deviceID)
	if err != nil {
		return nil, fmt.Errorf("tx.GetAllByDeviceId: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return lo.Map(result, func(r models.Message, indx int) ReportGetAllByDeviceId {
		if dev, ok := deviceByDeviceID[r.DeviceId]; ok {
			return ReportGetAllByDeviceId{
				DeviceID:    r.DeviceId,
				Name:        dev.Name,
				DeviceType:  dev.DeviceType,
				Address:     dev.Address,
				Responsible: dev.Responsible,
				Message:     r.Message,
				MessageType: r.MessageType,
				GotAt:       r.GotAt,
			}
		}
		return ReportGetAllByDeviceId{
			DeviceID:    r.DeviceId,
			Name:        defaultDevice.Name,
			DeviceType:  defaultDevice.DeviceType,
			Address:     defaultDevice.Address,
			Responsible: defaultDevice.Responsible,
			Message:     r.Message,
			MessageType: r.MessageType,
			GotAt:       r.GotAt,
		}
	}), nil
}

type ReportGetCountByMessageType struct {
	DeviceID    int32
	Name        string
	DeviceType  string
	Address     string
	Responsible []int32
	Count       int32
}

func (ms *MessagesService) GetCountByMessageType(messageType string) ([]ReportGetCountByMessageType, error) {
	tx, err := ms.messageRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return nil, fmt.Errorf("ms.messageRepo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.GetCountByMessageType(messageType)
	if err != nil {
		return nil, fmt.Errorf("tx.GetCountByMessageType: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return lo.Map(result.Count, func(r models.CountByDeviceID, indx int) ReportGetCountByMessageType {
		if dev, ok := deviceByDeviceID[r.DeviceId]; ok {
			return ReportGetCountByMessageType{
				DeviceID:    r.DeviceId,
				Name:        dev.Name,
				DeviceType:  dev.DeviceType,
				Address:     dev.Address,
				Responsible: dev.Responsible,
				Count:       r.Count,
			}
		}
		return ReportGetCountByMessageType{
			DeviceID:    r.DeviceId,
			Name:        defaultDevice.Name,
			DeviceType:  defaultDevice.DeviceType,
			Address:     defaultDevice.Address,
			Responsible: defaultDevice.Responsible,
			Count:       r.Count,
		}
	}), nil
}

func (ms *MessagesService) MonthReport() ([]models.MonthReportRow, error) {
	tx, err := ms.messageRepo.BeginTx(context.Background())
	if err != nil {
		ms.log.Error("tx.BeginTx", zap.Error(err))
		return nil, fmt.Errorf("ms.messageRepo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.MonthReport()
	if err != nil {
		return nil, fmt.Errorf("tx.MonthReport: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}
	return result, nil
}
