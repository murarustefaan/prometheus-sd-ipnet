package main

import (
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/murarustefaan/prometheus-sd-ipnet/pkg/http"
	"github.com/murarustefaan/prometheus-sd-ipnet/pkg/scanner"
)

func main() {
	network := flag.String("network", "", "Network in CIDR notation (e.g., 192.168.1.0/24)")
	port := flag.Int("port", 9100, "Target port to scan")
	timeout := flag.Duration("timeout", 2*time.Second, "Timeout for individual scans")
	interval := flag.Duration("interval", 5*time.Minute, "Interval between rescans")
	concurrency := flag.Int("concurrency", 100, "Number of concurrent scans")
	listenAddr := flag.String("listen-addr", ":8080", "HTTP server listen address")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")

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
		log.Fatal().Msg("Network is required. Use -network flag (e.g., -network 192.168.1.0/24)")
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
