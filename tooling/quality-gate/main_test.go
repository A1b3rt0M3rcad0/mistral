package main

import "testing"

func TestLayerViolation(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		dependency string
		want       bool
	}{
		{"domain to domain", "packages/mistral-core/modules/dungeon/domain/engine.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain", false},
		{"domain to application", "packages/mistral-core/modules/dungeon/domain/engine.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application", true},
		{"application to domain", "packages/mistral-core/modules/dungeon/application/service.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain", false},
		{"application to application port", "packages/mistral-core/modules/dungeon/application/boss.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application", false},
		{"application to infra", "packages/mistral-core/modules/gathering/application/service.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory", true},
		{"infra to shared infra", "packages/mistral-core/modules/inventory/infra/memory/repository.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory", false},
		{"infra to composition", "packages/mistral-core/modules/inventory/infra/memory/repository.go", "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/composition", true},
		{"stdlib ignored", "packages/mistral-core/modules/dungeon/domain/engine.go", "time", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := layerViolation(test.file, test.dependency) != ""
			if got != test.want {
				t.Fatalf("layerViolation(%q, %q) violation=%v, want %v", test.file, test.dependency, got, test.want)
			}
		})
	}
}
