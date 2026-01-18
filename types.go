package config

import (
	"io"
	"time"
)

type Environment string

const (
	Local       Environment = "local"
	Development Environment = "development"
	Production  Environment = "production"
)

type InitParams struct {
	Environment Environment
	Watcher     WatcherParams
}

type WatcherParams struct {
	IsEnabled bool
	Logger    io.Writer
	Interval  time.Duration
}
