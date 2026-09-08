package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	charactermemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/infra/memory"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
)

func TestCharacterQueriesRequireOwnershipAndReturnAuthoritativeState(t *testing.T) {
	characters := charactermemory.NewRepository()
	playerCharacter, err := character.New("hero-1", "human", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := characters.Create(context.Background(), playerCharacter); err != nil {
		t.Fatal(err)
	}

	inventories := inventorymemory.NewRepository()
	playerInventory, err := inventory.New("hero-1")
	if err != nil {
		t.Fatal(err)
	}
	acquiredAt := time.Date(2026, 9, 8, 7, 0, 0, 0, time.UTC)
	if err := playerInventory.Add("iron_ore", 3, acquiredAt, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := inventories.Create(context.Background(), playerInventory); err != nil {
		t.Fatal(err)
	}

	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
		WithCharacterReader(characters),
		WithInventoryReader(inventories),
	)

	for path, expected := range map[string]string{
		"/api/v1/characters/hero-1":           `"id":"hero-1"`,
		"/api/v1/characters/hero-1/inventory": `"item_id":"iron_ore"`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200; body=%s", path, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("%s body = %s, want %s", path, recorder.Body.String(), expected)
		}
	}
}

func TestCharacterQueriesRejectNonOwnerBeforeReadingState(t *testing.T) {
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-2"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
	)
	for _, path := range []string{
		"/api/v1/characters/hero-1",
		"/api/v1/characters/hero-1/inventory",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want 403", path, recorder.Code)
		}
	}
}

func TestCharacterQueriesRequireAuthentication(t *testing.T) {
	server := New(content.NewRegistry())
	for _, path := range []string{
		"/api/v1/characters/hero-1",
		"/api/v1/characters/hero-1/inventory",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, recorder.Code)
		}
	}
}
