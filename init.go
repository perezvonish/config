package config

import (
	"context"
	"fmt"
	"os"
)

func Init(ctx context.Context, params InitParams) (*Config, error) {
	filePath := fmt.Sprintf(".env.%v", params.Environment)

	logger := params.Watcher.Logger
	if logger == nil {
		logger = os.Stdout
	}

	cfg := &Config{
		env:      params.Environment,
		filePath: filePath,
		data:     make(map[string]any),
		logger:   logger,
	}

	if err := cfg.reload(); err != nil {
		return nil, fmt.Errorf("initial load failed: %w", err)
	}

	if params.Watcher.IsEnabled {
		go cfg.startWatchingUpdates(ctx, params.Watcher)
	}

	return cfg, nil
}
