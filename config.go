package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	sync.RWMutex
	filePath    string
	lastModTime time.Time
	logger      io.Writer
	env         Environment
	data        map[string]any
}

func (c *Config) GetEnv(key string) any {
	c.RLock()
	defer c.RUnlock()
	return c.data[key]
}

func (c *Config) SetEnv(key string, value any) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = value
}

func (c *Config) parseFile() (map[string]any, error) {
	file, err := os.Open(c.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	newData := make(map[string]any)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			newData[key] = value
		}
	}
	return newData, scanner.Err()
}

func (c *Config) reload() error {
	fileInfo, err := os.Stat(c.filePath)
	if err != nil {
		return err
	}

	newData, err := c.parseFile()
	if err != nil {
		return err
	}

	c.Lock()
	c.data = newData
	c.lastModTime = fileInfo.ModTime()
	c.Unlock()

	return nil
}

func (c *Config) startWatchingUpdates(ctx context.Context, params WatcherParams) {
	interval := params.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	fmt.Fprintln(c.logger, "Watcher started")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(c.logger, "Watcher stopped by context")
			return
		case <-ticker.C:
			fileInfo, err := os.Stat(c.filePath)
			if err != nil {
				fmt.Fprintf(c.logger, "Watcher error: %v\n", err)
				continue
			}

			if fileInfo.ModTime().After(c.lastModTime) {
				fmt.Fprintln(c.logger, "Detected file change, reloading...")
				if err := c.reload(); err != nil {
					fmt.Fprintf(c.logger, "Update failed: %v\n", err)
				} else {
					fmt.Fprintln(c.logger, "Config successfully updated")
				}
			}
		}
	}
}
