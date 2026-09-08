package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestContentRelease(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "mistral-content", Version: "0.1.0", Hash: "abc"}
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: "Human"}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/content/release", nil)
	New(registry).Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"version":"0.1.0"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
