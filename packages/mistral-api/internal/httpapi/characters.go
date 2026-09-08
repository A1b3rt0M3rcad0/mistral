package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type CharacterRegistrar interface {
	Execute(context.Context, characterapplication.RegistrationCommand) (characterapplication.RegistrationResult, error)
}

func WithCharacterRegistrar(registrar CharacterRegistrar) Option {
	return func(server *Server) {
		server.characterRegistrar = registrar
	}
}

type createCharacterRequest struct {
	RaceID string `json:"race_id"`
}

func (s *Server) createCharacter(w http.ResponseWriter, r *http.Request) {
	principal, err := s.resolvePrincipal(r)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}
	if s.characterRegistrar == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "character registration unavailable"})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key header is required"})
		return
	}

	var request createCharacterRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	request.RaceID = strings.TrimSpace(request.RaceID)
	if request.RaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "race_id is required"})
		return
	}

	result, err := s.characterRegistrar.Execute(r.Context(), characterapplication.RegistrationCommand{
		IdempotencyKey: idempotencyKey,
		SubjectID:      principal.SubjectID,
		RaceID:         request.RaceID,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, characterapplication.ErrUnknownRace):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "unknown race"})
		case errors.Is(err, persistence.ErrAlreadyExists), errors.Is(err, persistence.ErrIdempotencyKeyReuse), errors.Is(err, persistence.ErrCommandInProgress):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "character registration conflict"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "character registration failed"})
		}
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}
