package domain

import "errors"

var (
	ErrInvalidArgument           = errors.New("invalid argument")
	ErrAdapterNotFound           = errors.New("adapter not found")
	ErrCapabilityUnavailable     = errors.New("adapter capability is not available")
	ErrResourceFormatUnavailable = errors.New("adapter resource format is not available")
	ErrAdapterUnavailable        = errors.New("adapter is unavailable")
	ErrAdapterTimeout            = errors.New("adapter call timed out")
	ErrCatalogUnavailable        = errors.New("catalog is unavailable")
)
