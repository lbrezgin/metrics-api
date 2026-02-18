// Package config provides structures and logic for agent and
// server configuration.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type AgentConfig struct {
	Addr           string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

type ServerConfig struct {
	Addr         string `env:"ADDRESS"`
	LogCfg       *LogConfig
	PersisterCfg *PersisterConfig
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL"`
	Type   string `env:"LOG_TYPE"`
	Output string `env:"LOG_OUTPUT"`
}

type PersisterConfig struct {
	StoreInterval   *int   `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         *bool  `env:"RESTORE"`
}

// Configs is a type constraint that limits [LoadConfig] to
// supported configuration types.
type Configs interface {
	AgentConfig | ServerConfig
}

// LoadConfig populates the provided config structure with values from
// environmental variables. It accepts only types defined in the
// Configs interface.
func LoadConfig[T Configs](cfg *T) error {
	err := env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return nil
}
