package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type recordingRegistrar struct {
	command characterapplication.RegistrationCommand
	result  characterapplication.RegistrationResult
	err     error
	calls   int
}

func (r *recordingRegistrar) Execute(_ context.Context, command characterapplication.RegistrationCommand) (characterapplication.RegistrationResult, error) {
	r.command = command
	r.calls++
	return r.result, r.err
}

func TestCreateCharacterUsesAuthenticatedSubjectAndServerGeneratedIdentity(t *testing.T) {
	registrar := &recordingRegistrar{result: characterapplication.RegistrationResult{Character: character.Character{ID: "generated-hero", RaceID: "human", Level: 1}}}
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterRegistrar(registrar),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"race_id":"human"}`))
	request.Header.Set("Idempotency-Key", "register-1")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", recorder.Code, recorder.Body.String())
	}
	if registrar.calls != 1 {
		t.Fatalf("registrar calls = %d, want 1", registrar.calls)
	}
	if registrar.command.SubjectID != "subject-1" || registrar.command.RaceID != "human" || registrar.command.IdempotencyKey != "register-1" {
		t.Fatalf("unexpected registration command: %#v", registrar.command)
	}
}

func TestCreateCharacterRejectsClientSuppliedIdentityFields(t *testing.T) {
	for name, body := range map[string]string{
		"subject":   `{"race_id":"human","subject_id":"attacker"}`,
		"character": `{"race_id":"human","character_id":"chosen-by-client"}`,
	} {
		t.Run(name, func(t *testing.T) {
			registrar := &recordingRegistrar{}
			server := New(content.NewRegistry(),
				WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
				WithCharacterRegistrar(registrar),
			)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(body))
			request.Header.Set("Idempotency-Key", "register-1")
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
			if registrar.calls != 0 {
				t.Fatalf("registrar calls = %d, want 0", registrar.calls)
			}
		})
	}
}

func TestCreateCharacterRequiresAuthenticationAndIdempotencyKey(t *testing.T) {
	registrar := &recordingRegistrar{}
	unauthenticated := New(content.NewRegistry(), WithCharacterRegistrar(registrar))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"race_id":"human"}`))
	request.Header.Set("Idempotency-Key", "register-1")
	recorder := httptest.NewRecorder()
	unauthenticated.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", recorder.Code)
	}

	authenticated := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterRegistrar(registrar),
	)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"race_id":"human"}`))
	recorder = httptest.NewRecorder()
	authenticated.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency status = %d, want 400", recorder.Code)
	}
}

func TestCreateCharacterReplayReturnsOK(t *testing.T) {
	registrar := &recordingRegistrar{result: characterapplication.RegistrationResult{Character: character.Character{ID: "generated-hero", RaceID: "human", Level: 1}, Replayed: true}}
	server := New(content.NewRegistry(),
		WithPrincipalResolver(staticPrincipalResolver{principal: Principal{SubjectID: "subject-1"}}),
		WithCharacterRegistrar(registrar),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"race_id":"human"}`))
	request.Header.Set("Idempotency-Key", "register-1")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}
