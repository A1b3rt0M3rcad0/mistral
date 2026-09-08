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
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	identitymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/infra/memory"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
)

func TestListCharactersUsesAuthenticatedSubjectOwnership(t *testing.T) {
	characters := charactermemory.NewRepository()
	for _, definition := range []struct {
		id   string
		race string
	}{
		{id: "hero-2", race: "elf"},
		{id: "hero-1", race: "human"},
		{id: "outsider", race: "orc"},
	} {
		playerCharacter, err := character.New(definition.id, definition.race, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := characters.Create(context.Background(), playerCharacter); err != nil {
			t.Fatal(err)
		}
	}
	ownership := identitymemory.NewRepository()
	for _, binding := range []struct {
		subjectID   string
		characterID string
	}{
		{subjectID: "subject-1", characterID: "hero-2"},
		{subjectID: "subject-1", characterID: "hero-1"},
		{subjectID: "subject-2", characterID: "outsider"},
	} {
		record, err := identity.NewOwnership(binding.subjectID, binding.characterID)
		if err != nil {
			t.Fatal(err)
		}
		if err := ownership.Bind(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}

	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithOwnershipReader(ownership),
		WithCharacterReader(characters),
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/characters", nil)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	firstIndex := strings.Index(body, `"id":"hero-1"`)
	secondIndex := strings.Index(body, `"id":"hero-2"`)
	if firstIndex < 0 || secondIndex < 0 || firstIndex > secondIndex {
		t.Fatalf("characters are missing or not deterministic: %s", body)
	}
	if strings.Contains(body, `"id":"outsider"`) {
		t.Fatalf("list leaked another subject's character: %s", body)
	}
}

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
		"/api/v1/characters",
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
