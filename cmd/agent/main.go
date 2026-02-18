package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/lbrezgin/telemetry/internal/agent"
	"github.com/lbrezgin/telemetry/internal/agent/stats"
	"github.com/lbrezgin/telemetry/internal/config"
)

func main() {
	cfg := &config.AgentConfig{}

	if err := godotenv.Load(); err != nil {
		log.Printf("env file wasn't load successfully: %v\n", err)
	}

	if err := config.LoadConfig(cfg); err != nil {
		log.Fatal(err)
	}

	if err := parseFlags(cfg); err != nil {
		log.Fatal(err)
	}

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.AgentConfig) error {
	runtimeStats := stats.NewRuntimeStats()

	client := resty.New()
	client.SetDebug(true)

	agent := agent.NewAgent(runtimeStats, client, cfg)

	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		close(stop)
	}()

	if err := agent.Start(stop); err != nil {
		return err
	}

	log.Println("stopping agent...")
	return nil
}
