package httpapi

import (
	"context"
	"errors"
	"net/http"

	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
)

var ErrUnauthenticated = errors.New("authentication required")

type Principal struct {
	SubjectID string
}

type PrincipalResolver interface {
	Resolve(*http.Request) (Principal, error)
}

type CharacterAuthorizer interface {
	Authorize(context.Context, string, string) error
}

func WithPrincipalResolver(resolver PrincipalResolver) Option {
	return func(server *Server) {
		server.principals = resolver
	}
}

func WithCharacterAuthorizer(authorizer CharacterAuthorizer) Option {
	return func(server *Server) {
		server.characterAuthorizer = authorizer
	}
}

func (s *Server) authorizeCharacter(r *http.Request, characterID string) (Principal, error) {
	if s.principals == nil {
		return Principal{}, ErrUnauthenticated
	}
	principal, err := s.principals.Resolve(r)
	if err != nil {
		return Principal{}, err
	}
	if principal.SubjectID == "" {
		return Principal{}, ErrUnauthenticated
	}
	if s.characterAuthorizer == nil {
		return Principal{}, errors.New("character authorizer is not configured")
	}
	if err := s.characterAuthorizer.Authorize(r.Context(), principal.SubjectID, characterID); err != nil {
		if errors.Is(err, identityapplication.ErrForbidden) {
			return Principal{}, identityapplication.ErrForbidden
		}
		return Principal{}, err
	}
	return principal, nil
}
