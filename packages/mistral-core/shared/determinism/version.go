package determinism

import "fmt"

type Version string

const (
	RulesV1 Version = "mistral.rules.v1"
	Current Version = RulesV1
)

func Canonical(version Version) (Version, error) {
	switch version {
	case "", RulesV1:
		return RulesV1, nil
	default:
		return "", fmt.Errorf("unsupported deterministic ruleset version %q", version)
	}
}
