package main

import (
	"flag"
	"fmt"

	"github.com/lbrezgin/telemetry/internal/config"
)

// Agent related configuration flags.
var (
	flagServerAddr string

	flagReportInterval int
	flagPollInterval   int
)

// Agent default configuration fields.
const (
	defaultServerAddr     = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
)

// parseFlags parses all flags and sets the corresponding config fields
// only if they currently hold zero values.
func parseFlags(cfg *config.Agent) error {
	flag.StringVar(&flagServerAddr, "a", defaultServerAddr, "server address (host:port)")

	flag.IntVar(&flagReportInterval, "r", defaultReportInterval, "metrics report interval")
	flag.IntVar(&flagPollInterval, "p", defaultPollInterval, "metrics poll interval")

	flag.Parse()

	setIfEmpty(&cfg.Addr, flagServerAddr)

	if err := setIf0(&cfg.ReportInterval, flagReportInterval); err != nil {
		return fmt.Errorf("report interval: %w", err)
	}

	if err := setIf0(&cfg.PollInterval, flagPollInterval); err != nil {
		return fmt.Errorf("poll interval: %w", err)
	}

	return nil
}

func validateDuration(flg int) error {
	if flg <= 0 {
		return fmt.Errorf("interval must be greater than zero, got: %d", flg)
	}
	return nil
}

func setIfEmpty(field *string, val string) {
	if *field == "" {
		*field = val
	}
}

func setIf0(field *int, val int) error {
	if *field == 0 {
		if err := validateDuration(val); err != nil {
			return err
		}
		*field = val
	}
	return nil
}
