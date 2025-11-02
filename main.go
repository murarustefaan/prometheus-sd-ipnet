package main

import (
	"flag"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/murarustefaan/prometheus-sd-ipnet/pkg/http"
	"github.com/murarustefaan/prometheus-sd-ipnet/pkg/scanner"
)

type Config struct {
	Network     string        `envconfig:"NETWORK"`
	Port        int           `envconfig:"PORT" default:"9100"`
	Timeout     time.Duration `envconfig:"TIMEOUT" default:"2s"`
	Interval    time.Duration `envconfig:"INTERVAL" default:"5m"`
	Concurrency int           `envconfig:"CONCURRENCY" default:"100"`
	ListenAddr  string        `envconfig:"LISTEN_ADDR" default:":8080"`
	LogLevel    string        `envconfig:"LOG_LEVEL" default:"info"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log := zerolog.New(os.Stdout).With().Timestamp().Logger()
		log.Fatal().Err(err).Msg("Failed to process environment variables")
	}

	network := flag.String("network", cfg.Network, "Network in CIDR notation (e.g., 192.168.1.0/24)")
	port := flag.Int("port", cfg.Port, "Target port to scan")
	timeout := flag.Duration("timeout", cfg.Timeout, "Timeout for individual scans")
	interval := flag.Duration("interval", cfg.Interval, "Interval between rescans")
	concurrency := flag.Int("concurrency", cfg.Concurrency, "Number of concurrent scans")
	listenAddr := flag.String("listen-addr", cfg.ListenAddr, "HTTP server listen address")
	logLevel := flag.String("log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")

	flag.Parse()

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	switch *logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Str("provided", *logLevel).Msg("Invalid log level, defaulting to info")
	}

	if *network == "" {
		log.Fatal().Msg("Network is required. Use -network flag or NETWORK environment variable (e.g., -network 192.168.1.0/24)")
	}
	log.Info().Msg("booting up")

	scanner, err := scanner.NewScanner(*network, *port, *timeout, *interval, *concurrency, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create scanner")
	}
	scanner.Start()

	manager := http.NewManager(scanner, *listenAddr, log)

	sem := make(chan error, 1)
	go func() {
		if err := manager.Start(); err != nil {
			sem <- err
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	err = <-sem
	if err != nil {
		log.Fatal().Err(err).Msg("exiting with server error")
	}
}
