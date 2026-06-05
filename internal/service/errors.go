package service

import "errors"

var (
	ErrAdminAlreadyInitialized = errors.New("admin already initialized")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrEmptyUsername           = errors.New("username must not be empty")
	ErrEmptyPassword           = errors.New("password must not be empty")
	ErrEmptyUserName           = errors.New("user name must not be empty")
	ErrInvalidExpiry           = errors.New("expiry time must not be zero")
	ErrInvalidDeviceSlotCount  = errors.New("device slot count must be greater than zero")
	ErrInvalidRenewalDays      = errors.New("renewal days must be greater than zero")
	ErrInvalidRenewalTarget    = errors.New("renewal target must be later than the current effective expiry")
)
