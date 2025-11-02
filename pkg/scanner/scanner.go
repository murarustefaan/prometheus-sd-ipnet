package scanner

import (
	"errors"
	"fmt"
	"iter"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Scanner struct {
	network     *net.IPNet
	port        int
	timeout     time.Duration
	interval    time.Duration
	concurrency int

	targets []string

	mu  sync.RWMutex
	log zerolog.Logger
}

func NewScanner(network string, port int, timeout, interval time.Duration, concurrency int, log zerolog.Logger) (*Scanner, error) {
	inet, err := validate(network)
	if err != nil {
		return nil, err
	}

	log = log.With().
		Str("component", "scanner").
		Str("network", inet.String()).
		Int("port", port).
		Logger()

	return &Scanner{
		network:     inet,
		port:        port,
		timeout:     timeout,
		interval:    interval,
		concurrency: concurrency,
		targets:     make([]string, 0),
		log:         log,
	}, nil
}

// validate checks if the provided network string is a valid CIDR or IP
// it also does not allow for public network scanning
func validate(network string) (*net.IPNet, error) {
	_, inet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, fmt.Errorf("invalid network format: %s", network)
	}

	private := inet.IP.IsPrivate()
	if !private {
		return nil, errors.New("only private networks are allowed for scanning")
	}

	return inet, nil
}

func (s *Scanner) iterate() iter.Seq[string] {
	ip, err := netip.ParseAddr(s.network.IP.String())
	if err != nil {
		s.log.Fatal().Err(err).Msg("failed to parse network IP")
	}

	return func(yield func(string) bool) {
		for ip.IsValid() && s.network.Contains(ip.AsSlice()) {
			if ip.IsLoopback() || ip.IsMulticast() {
				ip.Next()
				continue
			}
			yield(ip.String())
			ip = ip.Next()
		}
	}
}

func (s *Scanner) scan() {
	startTime := time.Now()
	s.log.Info().Msg("starting network scan")

	eg := &errgroup.Group{}
	eg.SetLimit(s.concurrency)

	targets := make([]string, 0)
	mu := sync.Mutex{}

	for address := range s.iterate() {
		log := s.log.With().Str("ip", address).Logger()
		log.Debug().Msg("scanning IP address")

		eg.Go(func() error {
			target := net.JoinHostPort(address, strconv.Itoa(s.port))

			conn, err := net.DialTimeout("tcp", target, s.timeout)
			if err != nil {
				return nil
			}
			_ = conn.Close()

			log.Info().Msg("discovered open port")
			mu.Lock()
			defer mu.Unlock()
			targets = append(targets, target)

			return nil
		})
	}
	_ = eg.Wait()

	s.mu.Lock()
	s.targets = targets
	s.mu.Unlock()

	duration := time.Since(startTime)
	s.log.Info().
		Int("found_targets", len(targets)).
		Dur("took", duration).
		Msg("network scan completed")
}

func (s *Scanner) Start() {
	s.scan()

	ticker := time.NewTicker(s.interval)
	go func() {
		for range ticker.C {
			s.scan()
		}
	}()
}

func (s *Scanner) Targets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.targets
}
