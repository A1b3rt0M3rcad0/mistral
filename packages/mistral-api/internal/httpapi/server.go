package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Option func(*Server)

type Server struct {
	registry content.Registry
	gameplay *gameplaydb.Store
	mux      *http.ServeMux
}

func WithGameplayStore(store *gameplaydb.Store) Option {
	return func(server *Server) {
		server.gameplay = store
	}
}

func New(registry content.Registry, options ...Option) *Server {
	s := &Server{registry: registry, mux: http.NewServeMux()}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /api/v1/content/release", s.contentRelease)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.gameplay == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "persistence": "disabled"})
		return
	}
	if err := s.gameplay.DB.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "persistence": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "persistence": "ready"})
}

func (s *Server) contentRelease(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       s.registry.Manifest.Name,
		"version":    s.registry.Manifest.Version,
		"hash":       s.registry.Manifest.Hash,
		"release_id": s.registry.Manifest.ReleaseID(),
		"races":      len(s.registry.Races),
		"items":      len(s.registry.Items),
		"monsters":   len(s.registry.Monsters),
		"dungeons":   len(s.registry.Dungeons),
		"recipes":    len(s.registry.Recipes),
		"gathering":  len(s.registry.Gathering),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
