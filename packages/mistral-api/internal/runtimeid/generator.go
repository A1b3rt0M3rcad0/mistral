package runtimeid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Generator struct{}

func (Generator) NewCharacterID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate character id: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
