package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestContentRacesReturnsCreationCatalogInDeterministicOrder(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "abc"}
	registry.Races["troll"] = content.RaceDefinition{ID: "troll", Name: "Troll", BaseModifiers: map[string]float64{"strength": 1.2}}
	registry.Races["elf"] = content.RaceDefinition{ID: "elf", Name: "Elf", BaseModifiers: map[string]float64{"agility": 1.1}}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/content/races", nil)
	New(registry).Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var response struct {
		ReleaseID string                   `json:"release_id"`
		Races     []content.RaceDefinition `json:"races"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.ReleaseID != "1@sha256:abc" {
		t.Fatalf("release id = %q", response.ReleaseID)
	}
	if len(response.Races) != 2 || response.Races[0].ID != "elf" || response.Races[1].ID != "troll" {
		t.Fatalf("unexpected races = %#v", response.Races)
	}
	if got := registry.Races["elf"].BaseModifiers["agility"]; got != 1.1 {
		t.Fatalf("registry mutated: agility = %v", got)
	}
}
