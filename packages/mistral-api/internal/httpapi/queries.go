package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	decayapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/application"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type CharacterReader interface {
	Get(context.Context, string) (persistence.Record[character.Character], error)
}

type InventoryReader interface {
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
}

type OwnershipReader interface {
	BySubject(context.Context, string) ([]identity.Ownership, error)
}

func WithCharacterReader(reader CharacterReader) Option {
	return func(server *Server) {
		server.characterReader = reader
	}
}

func WithInventoryReader(reader InventoryReader) Option {
	return func(server *Server) {
		server.inventoryReader = reader
	}
}

func WithOwnershipReader(reader OwnershipReader) Option {
	return func(server *Server) {
		server.ownershipReader = reader
	}
}

func (s *Server) listCharacters(w http.ResponseWriter, r *http.Request) {
	principal, err := s.resolvePrincipal(r)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}
	if s.ownershipReader == nil || s.characterReader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "character list unavailable"})
		return
	}
	ownerships, err := s.ownershipReader.BySubject(r.Context(), principal.SubjectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "character list failed"})
		return
	}
	characters := make([]character.Character, 0, len(ownerships))
	for _, ownership := range ownerships {
		record, err := s.characterReader.Get(r.Context(), ownership.CharacterID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "character list is inconsistent"})
			return
		}
		characters = append(characters, record.Value)
	}
	sort.Slice(characters, func(left, right int) bool {
		return characters[left].ID < characters[right].ID
	})
	writeJSON(w, http.StatusOK, map[string]any{"characters": characters})
}

func (s *Server) getCharacter(w http.ResponseWriter, r *http.Request) {
	characterID := strings.TrimSpace(r.PathValue("characterID"))
	if characterID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "character id is required"})
		return
	}
	if !s.authorizeCharacterRead(w, r, characterID) {
		return
	}
	if s.characterReader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "character query unavailable"})
		return
	}
	record, err := s.characterReader.Get(r.Context(), characterID)
	if err != nil {
		if errors.Is(err, persistence.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "character not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "character query failed"})
		return
	}
	writeJSON(w, http.StatusOK, record.Value)
}

func (s *Server) getInventory(w http.ResponseWriter, r *http.Request) {
	characterID := strings.TrimSpace(r.PathValue("characterID"))
	if characterID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "character id is required"})
		return
	}
	if !s.authorizeCharacterRead(w, r, characterID) {
		return
	}
	if s.inventoryReader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory query unavailable"})
		return
	}
	record, err := s.inventoryReader.Get(r.Context(), characterID)
	if err != nil {
		if errors.Is(err, persistence.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "inventory not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inventory query failed"})
		return
	}
	resolved := record.Value.Clone()
	if _, err := decayapplication.NewService(s.registry).Resolve(&resolved, time.Now().UTC()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inventory projection failed"})
		return
	}
	writeJSON(w, http.StatusOK, resolved)
}

func (s *Server) authorizeCharacterRead(w http.ResponseWriter, r *http.Request, characterID string) bool {
	if _, err := s.authorizeCharacter(r, characterID); err != nil {
		switch {
		case errors.Is(err, ErrUnauthenticated):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		case errors.Is(err, identityapplication.ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "character access forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authorization failed"})
		}
		return false
	}
	return true
}
