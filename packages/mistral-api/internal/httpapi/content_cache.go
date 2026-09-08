package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

func (s *Server) contentETag() string {
	sum := sha256.Sum256([]byte(s.registry.Manifest.ReleaseID()))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func (s *Server) contentNotModified(w http.ResponseWriter, r *http.Request) bool {
	etag := s.contentETag()
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")
	for _, candidate := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}
