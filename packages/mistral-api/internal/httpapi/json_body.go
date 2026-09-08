package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxRequestBodyBytes int64 = 1 << 20

var (
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrRequestBodyTooLarge = errors.New("request body too large")
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	if r == nil || r.Body == nil {
		return ErrInvalidRequestBody
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return classifyJSONBodyError(err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%w: request body must contain one JSON object", ErrInvalidRequestBody)
		}
		return classifyJSONBodyError(err)
	}
	return nil
}

func classifyJSONBodyError(err error) error {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return ErrRequestBodyTooLarge
	}
	return fmt.Errorf("%w: %v", ErrInvalidRequestBody, err)
}

func writeJSONBodyError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrRequestBodyTooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
}
