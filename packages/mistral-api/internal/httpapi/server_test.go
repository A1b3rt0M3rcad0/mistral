package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type readinessFunc func(context.Context) error

func (fn readinessFunc) Ready(ctx context.Context) error { return fn(ctx) }

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

func TestReadyReportsPersistenceDisabledWithoutGameplayStore(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	New(content.NewRegistry()).Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"persistence":"disabled"`) {
		t.Fatalf("status/body = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestReadyRejectsGameplayStoreWhenReadinessCheckFails(t *testing.T) {
	server := New(content.NewRegistry(), func(server *Server) {
		server.gameplay = &gameplaydb.Store{}
		server.readiness = readinessFunc(func(context.Context) error { return errors.New("schema unavailable") })
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"persistence":"unavailable"`) {
		t.Fatalf("status/body = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestReadyAcceptsGameplayStoreOnlyAfterReadinessCheckPasses(t *testing.T) {
	server := New(content.NewRegistry(), func(server *Server) {
		server.gameplay = &gameplaydb.Store{}
		server.readiness = readinessFunc(func(context.Context) error { return nil })
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"persistence":"ready"`) {
		t.Fatalf("status/body = %d %s", recorder.Code, recorder.Body.String())
	}
}
