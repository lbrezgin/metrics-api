package main

import (
	"flag"

	"github.com/lbrezgin/telemetry/internal/config"
)

var (
	// Server related configuration flags.
	flagRunAddr string

	// Logger related configuration flags.
	flagLogLevel  string
	flagLogType   string
	flagLogOutput string

	// Persister related configuration fields.
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
)

const (
	// Server default configuration fields.
	defaultRunAddr = "localhost:8080"

	// Logger default configuration fields.
	defaultLogLevel  = "info"
	defaultLogType   = "text"
	defaultLogOutput = "stdout"

	// Persister default configuration fields.
	defaultStoreInterval   = 300
	defaultFileStoragePath = "snapshot.json"
	defaultRestore         = false
)

// parseFlags parses all flags and sets the corresponding config fields
// only if they currently hold zero values.
func parseFlags(cfg *config.ServerConfig) {
	flag.StringVar(&flagRunAddr, "a", defaultRunAddr, "server address (host:port)")

	flag.StringVar(&flagLogLevel, "ll", defaultLogLevel, "log level (debug | info | warn | error)")
	flag.StringVar(&flagLogType, "lt", defaultLogType, "log format (text | json)")
	flag.StringVar(&flagLogOutput, "lo", defaultLogOutput, "log output (stdout | file | both)")

	flag.IntVar(&flagStoreInterval, "i", defaultStoreInterval, "interval for persisting server metrics (0 - synchronously)")
	flag.StringVar(&flagFileStoragePath, "f", defaultFileStoragePath, "file path for storing and restoring metrics")
	flag.BoolVar(&flagRestore, "r", defaultRestore, "restore metrics from file on startup")
	flag.Parse()

	// If configuration fields weren't filled before, we set them
	// using flag values. If flags weren't provided either,
	// default values will be used.

	setIfEmpty(&cfg.Addr, flagRunAddr)
	// Additional checks to ensure LogConfig is initialized.
	if cfg.LogCfg == nil {
		cfg.LogCfg = &config.LogConfig{}
	}

	setIfEmpty(&cfg.LogCfg.Level, flagLogLevel)
	setIfEmpty(&cfg.LogCfg.Type, flagLogType)
	setIfEmpty(&cfg.LogCfg.Output, flagLogOutput)

	// Additional check to ensure PersisterConfig is initialized.
	if cfg.PersisterCfg == nil {
		cfg.PersisterCfg = &config.PersisterConfig{}
	}

	setIfNil(&cfg.PersisterCfg.StoreInterval, flagStoreInterval)
	setIfEmpty(&cfg.PersisterCfg.FileStoragePath, flagFileStoragePath)
	setIfNil(&cfg.PersisterCfg.Restore, flagRestore)
}

func setIfEmpty(field *string, val string) {
	if field == nil {
		panic("config: field pointer shouldn't be nil")
	}

	if *field == "" {
		*field = val
	}
}

func setIfNil[T any](field **T, val T) {
	if field == nil {
		panic("config: pointer to field pointer must not be nil")
	}

	if *field == nil {
		*field = &val
	}
}
