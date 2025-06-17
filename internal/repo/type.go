package repo

import (
	"monolith/internal/models"
	"time"
)

type CreateUserOpts struct {
	Name     string
	Email    string
	Password string
}

type CreateUserResult struct {
	ID    int32
	Name  string
	Email string
}

type UpdateUsersOpts struct {
	ID       int32
	Name     *string
	Email    *string
	Password *string
}

type ReadUsersResult struct {
	Users []models.User
}

type AuthorizeOpts struct {
	Email    string
	Password string
}

type UpdateDeviceOpts struct {
	ID          int32
	Name        *string
	DeviceType  *string
	Address     *string
	Responsible []int32
}

type ReadDevicesResult struct {
	Devices []models.Device
}

type UpdateTagsOpts struct {
	ID            int32
	Name          *string
	DeviceId      *int32
	Regexp        *string
	CompareType   *string
	Value         *string
	ArrayIndex    *int32
	Subject       *string
	SeverityLevel *string
}

type ReadTagsResult struct {
	Tags []models.Tag
}

type MessagesGetAllByPeriodOpts struct {
	StartTime time.Time
	EndTime   time.Time
}

type GetCountByMessageTypeResult struct {
	Count []models.CountByDeviceID
}
