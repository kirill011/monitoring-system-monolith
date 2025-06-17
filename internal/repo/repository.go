package repo

import (
	"context"
	"monolith/internal/models"
)

type Auth interface {
	BeginTx(ctx context.Context) (Auth, error)
	Commit() error
	Rollback() error

	Create(opts CreateUserOpts) (CreateUserResult, error)
	Read(ctx context.Context) (ReadUsersResult, error)
	Update(ctx context.Context, opts UpdateUsersOpts) error
	Delete(ctx context.Context, id int32) error
	Authorize(opts AuthorizeOpts) (int, error)
	GetEmailsByIDs(ctx context.Context, userID []int32) ([]string, error)
}

type Devices interface {
	BeginTx(ctx context.Context) (Devices, error)
	Commit() error
	Rollback() error

	Create(opts models.Device) (models.Device, error)
	Read(ctx context.Context) (ReadDevicesResult, error)
	Update(ctx context.Context, opts UpdateDeviceOpts) error
	Delete(ctx context.Context, id int32) error
	GetResponsible(ctx context.Context, deviceID int32) ([]int32, error)
}

type Tags interface {
	BeginTx(ctx context.Context) (Tags, error)
	Commit() error
	Rollback() error

	Create(opts models.Tag) (models.Tag, error)
	Read(ctx context.Context) (ReadTagsResult, error)
	Update(ctx context.Context, opts UpdateTagsOpts) error
	Delete(ctx context.Context, id int32) error
}

type Messages interface {
	BeginTx(ctx context.Context) (Messages, error)
	Commit() error
	Rollback() error

	Create(opts models.Message) error
	GetAllByPeriod(opts MessagesGetAllByPeriodOpts) ([]models.Message, error)
	GetAllByDeviceId(deviceID int32) ([]models.Message, error)
	GetCountByMessageType(messageType string) (GetCountByMessageTypeResult, error)
	MonthReport() ([]models.MonthReportRow, error)
}
