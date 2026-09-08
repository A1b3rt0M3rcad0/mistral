package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
)

type staticPrincipalResolver struct {
	principal Principal
	err       error
}

func (r staticPrincipalResolver) Resolve(*http.Request) (Principal, error) {
	return r.principal, r.err
}

type staticCharacterAuthorizer struct {
	subjectID   string
	characterID string
}

func (a staticCharacterAuthorizer) Authorize(_ context.Context, subjectID, characterID string) error {
	if subjectID != a.subjectID || characterID != a.characterID {
		return identityapplication.ErrForbidden
	}
	return nil
}

func TestAuthorizeCharacterSeparatesAuthenticationFromOwnership(t *testing.T) {
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterAuthorizer(staticCharacterAuthorizer{subjectID: "subject-1", characterID: "hero-1"}),
	)
	request := httptest.NewRequest("POST", "/", nil)
	principal, err := server.authorizeCharacter(request, "hero-1")
	if err != nil {
		t.Fatal(err)
	}
	if principal.SubjectID != "subject-1" {
		t.Fatalf("subject = %q, want subject-1", principal.SubjectID)
	}
	if _, err := server.authorizeCharacter(request, "hero-2"); !errors.Is(err, identityapplication.ErrForbidden) {
		t.Fatalf("non-owner should be forbidden, got %v", err)
	}
}

func TestAuthorizeCharacterRequiresPrincipalResolver(t *testing.T) {
	server := New(content.NewRegistry())
	request := httptest.NewRequest("POST", "/", nil)
	if _, err := server.authorizeCharacter(request, "hero-1"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}
