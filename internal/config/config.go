package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Agent struct {
	Addr           string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

type Server struct {
	Addr         string `env:"ADDRESS"`
	RepoCfg      *Repo
	LogCfg       *Log
	PersisterCfg *Persister
}

type Repo struct {
	DSN            string `env:"DATABASE_DSN"`
	Driver         string `env:"DRIVER" envDefault:"pgx"`
	MigrationsPath string `env:"MIGRATIONS_PATH" envDefault:"file://migrations"`
}

type Log struct {
	Level  string `env:"LOG_LEVEL"`
	Type   string `env:"LOG_TYPE"`
	Output string `env:"LOG_OUTPUT"`
}

type Persister struct {
	StoreInterval   *int   `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         *bool  `env:"RESTORE"`
}

// Configs is a type constraint that limits [Load] to
// supported configuration types.
type Configs interface {
	Agent | Server
}

// Load populates the provided config structure with values from
// environmental variables. It accepts only types defined in the
// Configs interface.
func Load[T Configs](cfg *T) error {
	err := env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return nil
}
