package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Option func(*Server)

type ReadinessChecker interface {
	Ready(context.Context) error
}

type Server struct {
	registry            content.Registry
	gameplay            *gameplaydb.Store
	readiness           ReadinessChecker
	principals          PrincipalResolver
	characterAuthorizer CharacterAuthorizer
	characterRegistrar  CharacterRegistrar
	characterCrafter    CharacterCrafter
	characterReader     CharacterReader
	inventoryReader     InventoryReader
	ownershipReader     OwnershipReader
	mux                 *http.ServeMux
}

func WithGameplayStore(store *gameplaydb.Store) Option {
	return func(server *Server) {
		server.gameplay = store
		server.readiness = store
		if store != nil {
			server.characterReader = store.Characters
			server.inventoryReader = store.Inventories
			server.ownershipReader = store.Ownership
		}
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
	s.mux.HandleFunc("GET /api/v1/content/races", s.contentRaces)
	s.mux.HandleFunc("GET /api/v1/characters", privateResponse(s.listCharacters))
	s.mux.HandleFunc("GET /api/v1/characters/{characterID}", privateResponse(s.getCharacter))
	s.mux.HandleFunc("GET /api/v1/characters/{characterID}/inventory", privateResponse(s.getInventory))
	s.mux.HandleFunc("POST /api/v1/characters", privateResponse(s.createCharacter))
	s.mux.HandleFunc("POST /api/v1/characters/{characterID}/crafts", privateResponse(s.craft))
}

func privateResponse(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		next(w, r)
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.gameplay == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "persistence": "disabled"})
		return
	}
	if s.readiness == nil || s.readiness.Ready(r.Context()) != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "persistence": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "persistence": "ready"})
}

func (s *Server) contentRelease(w http.ResponseWriter, r *http.Request) {
	if s.contentNotModified(w, r) {
		return
	}
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
		"decay":      len(s.registry.Decay),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
