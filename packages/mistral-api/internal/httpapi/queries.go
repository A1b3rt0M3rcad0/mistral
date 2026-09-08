package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type CharacterReader interface {
	Get(context.Context, string) (persistence.Record[character.Character], error)
}

type InventoryReader interface {
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
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
	writeJSON(w, http.StatusOK, record.Value)
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
