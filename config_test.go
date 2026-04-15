package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type testConfig struct {
	Host     string `env:"TEST_CFG_HOST" envDefault:"localhost"`
	Port     int    `env:"TEST_CFG_PORT" envDefault:"8080"`
	LogLevel string `env:"TEST_CFG_LOG_LEVEL"`
}

type nestedConfig struct {
	App testConfig
	DB  struct {
		DSN string `env:"TEST_CFG_DB_DSN" required:"true"`
	}
}

type requiredConfig struct {
	Secret string `env:"TEST_CFG_SECRET" required:"true"`
}

type unsupportedConfig struct {
	Flag bool `env:"TEST_CFG_FLAG"`
}

type unexportedConfig struct {
	exported   string `env:"TEST_CFG_UNEXPORTED"`
	NoEnvField string
}

type invalidIntConfig struct {
	Port int `env:"TEST_CFG_BAD_PORT"`
}

func clearTestEnv() {
	for _, key := range []string{
		"TEST_CFG_HOST", "TEST_CFG_PORT", "TEST_CFG_LOG_LEVEL",
		"TEST_CFG_SECRET", "TEST_CFG_FLAG", "TEST_CFG_UNEXPORTED",
		"TEST_CFG_BAD_PORT", "TEST_CFG_DB_DSN",
	} {
		os.Unsetenv(key)
	}
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		envFile   string
		presetEnv map[string]string
		cfg       interface{}
		opts      []Option
		wantErr   error
		validate  func(t *testing.T, cfg interface{})
	}{
		{
			name:    "defaults when env is empty",
			envFile: "",
			cfg:     &testConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*testConfig)
				if c.Host != "localhost" {
					t.Errorf("Host = %q, want %q", c.Host, "localhost")
				}
				if c.Port != 8080 {
					t.Errorf("Port = %d, want %d", c.Port, 8080)
				}
				if c.LogLevel != "" {
					t.Errorf("LogLevel = %q, want %q", c.LogLevel, "")
				}
			},
		},
		{
			name:    "values from env file",
			envFile: "TEST_CFG_HOST=example.com\nTEST_CFG_PORT=3000\nTEST_CFG_LOG_LEVEL=debug",
			cfg:     &testConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*testConfig)
				if c.Host != "example.com" {
					t.Errorf("Host = %q, want %q", c.Host, "example.com")
				}
				if c.Port != 3000 {
					t.Errorf("Port = %d, want %d", c.Port, 3000)
				}
				if c.LogLevel != "debug" {
					t.Errorf("LogLevel = %q, want %q", c.LogLevel, "debug")
				}
			},
		},
		{
			name:    "env file with comments and empty lines",
			envFile: "# comment\n\nTEST_CFG_HOST=fromfile\nINVALID_LINE\n",
			cfg:     &testConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*testConfig)
				if c.Host != "fromfile" {
					t.Errorf("Host = %q, want %q", c.Host, "fromfile")
				}
			},
		},
		{
			name:    "env file with quoted values",
			envFile: `TEST_CFG_HOST="quoted-host"` + "\n" + `TEST_CFG_LOG_LEVEL='single'`,
			cfg:     &testConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*testConfig)
				if c.Host != "quoted-host" {
					t.Errorf("Host = %q, want %q", c.Host, "quoted-host")
				}
				if c.LogLevel != "single" {
					t.Errorf("LogLevel = %q, want %q", c.LogLevel, "single")
				}
			},
		},
		{
			name:      "real env takes precedence over file",
			envFile:   "TEST_CFG_HOST=from-file",
			presetEnv: map[string]string{"TEST_CFG_HOST": "from-env"},
			cfg:       &testConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*testConfig)
				if c.Host != "from-env" {
					t.Errorf("Host = %q, want %q", c.Host, "from-env")
				}
			},
		},
		{
			name:    "nested structs",
			envFile: "TEST_CFG_HOST=nested-host\nTEST_CFG_PORT=9090\nTEST_CFG_DB_DSN=postgres://localhost",
			cfg:     &nestedConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*nestedConfig)
				if c.App.Host != "nested-host" {
					t.Errorf("App.Host = %q, want %q", c.App.Host, "nested-host")
				}
				if c.App.Port != 9090 {
					t.Errorf("App.Port = %d, want %d", c.App.Port, 9090)
				}
				if c.DB.DSN != "postgres://localhost" {
					t.Errorf("DB.DSN = %q, want %q", c.DB.DSN, "postgres://localhost")
				}
			},
		},
		{
			name:    "required field missing",
			envFile: "",
			cfg:     &requiredConfig{},
			wantErr: ErrLoadFields,
		},
		{
			name:      "required field present",
			envFile:   "",
			presetEnv: map[string]string{"TEST_CFG_SECRET": "s3cret"},
			cfg:       &requiredConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*requiredConfig)
				if c.Secret != "s3cret" {
					t.Errorf("Secret = %q, want %q", c.Secret, "s3cret")
				}
			},
		},
		{
			name:    "unsupported field type",
			envFile: "TEST_CFG_FLAG=true",
			cfg:     &unsupportedConfig{},
			wantErr: ErrLoadFields,
		},
		{
			name:    "invalid int value",
			envFile: "TEST_CFG_BAD_PORT=not_a_number",
			cfg:     &invalidIntConfig{},
			wantErr: ErrLoadFields,
		},
		{
			name:    "non-pointer config",
			envFile: "",
			cfg:     testConfig{},
			wantErr: ErrLoadFields,
		},
		{
			name:    "pointer to non-struct",
			envFile: "",
			cfg:     new(string),
			wantErr: ErrLoadFields,
		},
		{
			name:    "file not found",
			envFile: "",
			opts:    []Option{WithPath("nonexistent/.env")},
			cfg:     &testConfig{},
			wantErr: ErrLoadFile,
		},
		{
			name:    "nested struct with error",
			envFile: "",
			cfg: &struct {
				Inner struct {
					Val bool `env:"TEST_CFG_NESTED_BOOL"`
				}
			}{},
			wantErr: ErrLoadFields,
		},
		{
			name:    "unexported and no-tag fields are skipped",
			envFile: "TEST_CFG_UNEXPORTED=value",
			cfg:     &unexportedConfig{},
			validate: func(t *testing.T, cfg interface{}) {
				c := cfg.(*unexportedConfig)
				if c.NoEnvField != "" {
					t.Errorf("NoEnvField = %q, want %q", c.NoEnvField, "")
				}
			},
		},
		{
			name:    "int field with empty value uses zero",
			envFile: "",
			cfg: &struct {
				Val int `env:"TEST_CFG_EMPTY_INT"`
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv()

			for k, v := range tt.presetEnv {
				os.Setenv(k, v)
			}

			opts := tt.opts
			if tt.wantErr != ErrLoadFile || len(opts) > 0 {
				if len(opts) == 0 && tt.envFile != "" {
					path := writeEnvFile(t, tt.envFile)
					opts = append(opts, WithPath(path))
				} else if len(opts) == 0 {
					path := writeEnvFile(t, "")
					opts = append(opts, WithPath(path))
				}
			}

			err := Load(tt.cfg, opts...)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, tt.cfg)
			}
		})
	}
}

func TestNewFieldRequiredError(t *testing.T) {
	err := NewFieldRequiredError("MY_VAR")
	if !errors.Is(err, ErrLoadFields) {
		t.Errorf("expected error to wrap ErrLoadFields")
	}
	expected := "error on config fields loading: MY_VAR is required but not set"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}
