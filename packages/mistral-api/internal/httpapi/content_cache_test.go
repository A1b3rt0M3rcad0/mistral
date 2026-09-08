package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestContentEndpointsSupportReleaseETagRevalidation(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "abc"}
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: "Human"}
	server := New(registry)

	for _, path := range []string{"/api/v1/content/release", "/api/v1/content/races"} {
		first := httptest.NewRecorder()
		server.Handler().ServeHTTP(first, httptest.NewRequest(http.MethodGet, path, nil))
		if first.Code != http.StatusOK {
			t.Fatalf("%s first status = %d, want 200", path, first.Code)
		}
		etag := first.Header().Get("ETag")
		if etag == "" {
			t.Fatalf("%s did not return ETag", path)
		}
		if got := first.Header().Get("Cache-Control"); got != "public, no-cache" {
			t.Fatalf("%s Cache-Control = %q", path, got)
		}

		second := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("If-None-Match", etag)
		server.Handler().ServeHTTP(second, request)
		if second.Code != http.StatusNotModified {
			t.Fatalf("%s revalidation status = %d, want 304", path, second.Code)
		}
		if second.Body.Len() != 0 {
			t.Fatalf("%s 304 body = %q, want empty", path, second.Body.String())
		}
	}
}

func TestContentETagChangesWithReleaseIdentity(t *testing.T) {
	firstRegistry := content.NewRegistry()
	firstRegistry.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "abc"}
	secondRegistry := content.NewRegistry()
	secondRegistry.Manifest = content.Manifest{Name: "mistral", Version: "2", Hash: "def"}

	first := New(firstRegistry).contentETag()
	second := New(secondRegistry).contentETag()
	if first == second {
		t.Fatalf("different releases produced the same ETag: %s", first)
	}
}
