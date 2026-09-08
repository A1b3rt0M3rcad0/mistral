package runtimeid

import (
	"encoding/hex"
	"testing"
)

func TestGeneratorCreatesOpaque128BitCharacterID(t *testing.T) {
	id, err := (Generator{}).NewCharacterID()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := hex.DecodeString(id)
	if err != nil {
		t.Fatalf("character id is not hexadecimal: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("character id bytes = %d, want 16", len(decoded))
	}
}
