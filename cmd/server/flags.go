package main

import (
	"flag"

	"github.com/lbrezgin/telemetry/internal/config"
)

var (
	// Server related configuration flags.
	flagRunAddr     string
	flagDatabaseDsn string

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
	defaultRunAddr     = "localhost:8080"
	defaultDatabaseDsn = ""

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
func parseFlags(cfg *config.Server) {
	flag.StringVar(&flagRunAddr, "a", defaultRunAddr, "server address (host:port)")
	flag.StringVar(&flagDatabaseDsn, "d", defaultDatabaseDsn, "database url")

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

	// Additional checks to ensure Database config  is initialized.
	if cfg.RepoCfg == nil {
		cfg.RepoCfg = &config.Repo{}
	}
	setIfEmpty(&cfg.RepoCfg.DSN, flagDatabaseDsn)

	// Additional checks to ensure Log config is initialized.
	if cfg.LogCfg == nil {
		cfg.LogCfg = &config.Log{}
	}
	setIfEmpty(&cfg.LogCfg.Level, flagLogLevel)
	setIfEmpty(&cfg.LogCfg.Type, flagLogType)
	setIfEmpty(&cfg.LogCfg.Output, flagLogOutput)

	// Additional check to ensure Persister config is initialized.
	if cfg.PersisterCfg == nil {
		cfg.PersisterCfg = &config.Persister{}
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
