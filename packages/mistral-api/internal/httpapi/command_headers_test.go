package httpapi

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireIdempotencyKeyEnforcesOpaqueKeyBounds(t *testing.T) {
	accepted := httptest.NewRequest("POST", "/", nil)
	accepted.Header.Set("Idempotency-Key", strings.Repeat("a", maxIdempotencyKeyBytes))
	key, err := requireIdempotencyKey(accepted)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != maxIdempotencyKeyBytes {
		t.Fatalf("accepted key length = %d", len(key))
	}

	tooLong := httptest.NewRequest("POST", "/", nil)
	tooLong.Header.Set("Idempotency-Key", strings.Repeat("b", maxIdempotencyKeyBytes+1))
	if _, err := requireIdempotencyKey(tooLong); !errors.Is(err, ErrIdempotencyKeyTooLong) {
		t.Fatalf("too-long key error = %v", err)
	}

	missing := httptest.NewRequest("POST", "/", nil)
	missing.Header.Set("Idempotency-Key", "   ")
	if _, err := requireIdempotencyKey(missing); !errors.Is(err, ErrIdempotencyKeyRequired) {
		t.Fatalf("blank key error = %v", err)
	}
}
