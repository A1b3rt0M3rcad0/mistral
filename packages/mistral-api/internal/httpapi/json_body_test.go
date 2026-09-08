package httpapi

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONBodyRejectsOversizedPayload(t *testing.T) {
	body := `{"race_id":"human"}` + strings.Repeat(" ", int(maxRequestBodyBytes))
	request := httptest.NewRequest("POST", "/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	var destination createCharacterRequest
	if err := decodeJSONBody(recorder, request, &destination); !errors.Is(err, ErrRequestBodyTooLarge) {
		t.Fatalf("oversized body error = %v, want ErrRequestBodyTooLarge", err)
	}
}

func TestDecodeJSONBodyRejectsUnknownFieldsAndMultipleObjects(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":    `{"race_id":"human","subject_id":"forged"}`,
		"multiple objects": `{"race_id":"human"}{"race_id":"elf"}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(body))
			recorder := httptest.NewRecorder()
			var destination createCharacterRequest
			if err := decodeJSONBody(recorder, request, &destination); !errors.Is(err, ErrInvalidRequestBody) {
				t.Fatalf("decode error = %v, want ErrInvalidRequestBody", err)
			}
		})
	}
}
