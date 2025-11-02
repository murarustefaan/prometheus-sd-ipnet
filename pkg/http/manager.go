package http

import (
	"encoding/json"
	"net/http"

	"github.com/murarustefaan/prometheus-sd-ipnet/pkg/scanner"
	"github.com/rs/zerolog"
)

type Manager struct {
	scanner *scanner.Scanner
	server  *http.Server
	log     zerolog.Logger
}

func NewManager(s *scanner.Scanner, listenAddr string, log zerolog.Logger) *Manager {
	manager := &Manager{
		scanner: s,
		log:     log.With().Str("component", "http").Logger(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", manager.handleHealth)
	mux.HandleFunc("/targets", manager.handleTargets)

	manager.server = &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	return manager
}

func (m *Manager) Start() error {
	m.log.Info().Str("address", m.server.Addr).Msg("starting HTTP server")
	return m.server.ListenAndServe()
}

func (m *Manager) handleHealth(w http.ResponseWriter, _ *http.Request) {
	response := struct {
		Ok bool `json:"ok"`
	}{
		Ok: true,
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		m.log.Error().Err(err).Msg("failed to write health response")
		return
	}
}

func (m *Manager) handleTargets(w http.ResponseWriter, _ *http.Request) {
	targets := m.scanner.Targets()

	response := struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}{
		Labels: map[string]string{
			"job": "ipnet_scanner",
		},
	}
	response.Targets = append(response.Targets, targets...)

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(targets)
	if err != nil {
		m.log.Error().Err(err).Msg("failed to write targets response")
		return
	}
}
