package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

const maxIdempotencyKeyBytes = 255

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyTooLong  = errors.New("idempotency key is too long")
)

func requireIdempotencyKey(r *http.Request) (string, error) {
	if r == nil {
		return "", ErrIdempotencyKeyRequired
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return "", ErrIdempotencyKeyRequired
	}
	if len(key) > maxIdempotencyKeyBytes {
		return "", ErrIdempotencyKeyTooLong
	}
	return key, nil
}

func writeIdempotencyKeyError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrIdempotencyKeyTooLong) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key must not exceed 255 bytes"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key header is required"})
}
