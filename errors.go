package config

import (
	"errors"
	"fmt"
)

var (
	ErrLoadFile   = errors.New("error on config file loading")
	ErrLoadFields = errors.New("error on config fields loading")
)

func NewFieldRequiredError(field string) error {
	return fmt.Errorf("%w: %s is required but not set", ErrLoadFields, field)
}
