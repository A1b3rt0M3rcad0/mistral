package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	craftingapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type CharacterCrafter interface {
	Execute(context.Context, craftingapplication.CraftCommand) (craftingapplication.CraftCommandResult, error)
}

func WithCharacterCrafter(crafter CharacterCrafter) Option {
	return func(server *Server) {
		server.characterCrafter = crafter
	}
}

type craftRequest struct {
	RecipeID string `json:"recipe_id"`
	Station  string `json:"station"`
	Crafts   int    `json:"crafts"`
}

func (s *Server) craft(w http.ResponseWriter, r *http.Request) {
	characterID := strings.TrimSpace(r.PathValue("characterID"))
	if characterID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "character id is required"})
		return
	}
	if _, err := s.authorizeCharacter(r, characterID); err != nil {
		switch {
		case errors.Is(err, ErrUnauthenticated):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		case errors.Is(err, identityapplication.ErrForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "character access forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authorization failed"})
		}
		return
	}
	if s.characterCrafter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "crafting unavailable"})
		return
	}
	idempotencyKey, err := requireIdempotencyKey(r)
	if err != nil {
		writeIdempotencyKeyError(w, err)
		return
	}

	var request craftRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	request.RecipeID = strings.TrimSpace(request.RecipeID)
	request.Station = strings.TrimSpace(request.Station)
	if request.RecipeID == "" || request.Station == "" || request.Crafts <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "recipe_id, station and a positive crafts value are required"})
		return
	}

	result, err := s.characterCrafter.Execute(r.Context(), craftingapplication.CraftCommand{
		IdempotencyKey: idempotencyKey,
		CharacterID:    characterID,
		RecipeID:       request.RecipeID,
		Station:        request.Station,
		Crafts:         request.Crafts,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, craftingapplication.ErrCraftRejected):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "craft rejected"})
		case errors.Is(err, persistence.ErrConflict), errors.Is(err, persistence.ErrIdempotencyKeyReuse), errors.Is(err, persistence.ErrCommandInProgress):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "crafting conflict"})
		case errors.Is(err, persistence.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "character inventory not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "crafting failed"})
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}
