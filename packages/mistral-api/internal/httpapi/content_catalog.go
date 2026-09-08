package httpapi

import (
	"net/http"
	"sort"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func (s *Server) contentRaces(w http.ResponseWriter, r *http.Request) {
	if s.contentNotModified(w, r) {
		return
	}
	races := make([]content.RaceDefinition, 0, len(s.registry.Races))
	for _, race := range s.registry.Races {
		copy := race
		copy.BaseModifiers = cloneModifiers(race.BaseModifiers)
		races = append(races, copy)
	}
	sort.Slice(races, func(left, right int) bool {
		return races[left].ID < races[right].ID
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"release_id": s.registry.Manifest.ReleaseID(),
		"races":      races,
	})
}

func cloneModifiers(source map[string]float64) map[string]float64 {
	if len(source) == 0 {
		return map[string]float64{}
	}
	clone := make(map[string]float64, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
