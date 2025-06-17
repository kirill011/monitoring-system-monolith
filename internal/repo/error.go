package repo

import "errors"

var (
	ErrUserExists    = errors.New("user already exists")
	ErrUsersNotFound = errors.New("user not found")

	ErrDeviceExists   = errors.New("device already exists")
	ErrDeviceNotFound = errors.New("device not found")
)
