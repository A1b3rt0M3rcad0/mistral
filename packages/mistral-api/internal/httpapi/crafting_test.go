package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	craftingapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
)

type recordingCrafter struct {
	command craftingapplication.CraftCommand
	result  craftingapplication.CraftCommandResult
	err     error
	calls   int
}

func (c *recordingCrafter) Execute(_ context.Context, command craftingapplication.CraftCommand) (craftingapplication.CraftCommandResult, error) {
	c.command = command
	c.calls++
	return c.result, c.err
}

func TestCraftUsesPathCharacterAfterOwnershipAuthorization(t *testing.T) {
	crafter := &recordingCrafter{}
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
		WithCharacterCrafter(crafter),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters/hero-1/crafts", strings.NewReader(`{"recipe_id":"iron_sword","station":"blacksmith","crafts":1}`))
	request.Header.Set("Idempotency-Key", "craft-1")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if crafter.calls != 1 {
		t.Fatalf("crafter calls = %d, want 1", crafter.calls)
	}
	if crafter.command.CharacterID != "hero-1" || crafter.command.RecipeID != "iron_sword" || crafter.command.Station != "blacksmith" || crafter.command.Crafts != 1 || crafter.command.IdempotencyKey != "craft-1" {
		t.Fatalf("unexpected craft command: %#v", crafter.command)
	}
}

func TestCraftRejectsCharacterOverrideInBody(t *testing.T) {
	crafter := &recordingCrafter{}
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
		WithCharacterCrafter(crafter),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters/hero-1/crafts", strings.NewReader(`{"character_id":"other","recipe_id":"iron_sword","station":"blacksmith","crafts":1}`))
	request.Header.Set("Idempotency-Key", "craft-1")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if crafter.calls != 0 {
		t.Fatalf("crafter calls = %d, want 0", crafter.calls)
	}
}

func TestCraftRejectsNonOwnerBeforeMutation(t *testing.T) {
	crafter := &recordingCrafter{}
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-2"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
		WithCharacterCrafter(crafter),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters/hero-1/crafts", strings.NewReader(`{"recipe_id":"iron_sword","station":"blacksmith","crafts":1}`))
	request.Header.Set("Idempotency-Key", "craft-1")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if crafter.calls != 0 {
		t.Fatalf("crafter calls = %d, want 0", crafter.calls)
	}
}
