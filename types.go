package config

const defaultEnvFile = ".env"

type Option func(*options)

type options struct {
	filePath string
}

func WithPath(path string) Option {
	return func(o *options) {
		o.filePath = path
	}
}
