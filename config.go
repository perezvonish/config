package config

import (
	"fmt"

	"github.com/perezvonish/config/internal"
)

func Load(cfg interface{}, opts ...Option) error {
	o := options{filePath: defaultEnvFile}
	for _, opt := range opts {
		opt(&o)
	}

	if err := internal.LoadEnvFile(o.filePath); err != nil {
		return fmt.Errorf("%w: %v", ErrLoadFile, err)
	}

	if err := internal.LoadFromEnv(cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrLoadFields, err)
	}

	return nil
}
