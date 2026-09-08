package domain

import "testing"

func TestCharacterDungeonProgressionIsIdempotent(t *testing.T) {
	character, err := New("hero", "human", []string{"abandoned_mine"})
	if err != nil {
		t.Fatal(err)
	}
	if !character.CanEnter("abandoned_mine", 1) {
		t.Fatal("tier 1 should be unlocked on creation")
	}
	if character.CanEnter("abandoned_mine", 2) {
		t.Fatal("tier 2 must not be unlocked before boss progression")
	}

	if err := character.AdvanceDungeon("abandoned_mine", 1); err != nil {
		t.Fatal(err)
	}
	if !character.CanEnter("abandoned_mine", 2) {
		t.Fatal("tier 2 should be unlocked after defeating tier 1")
	}
	if err := character.AdvanceDungeon("abandoned_mine", 1); err != nil {
		t.Fatal(err)
	}
	if got := character.MaxUnlockedTier("abandoned_mine"); got != 2 {
		t.Fatalf("expected idempotent progression at tier 2, got %d", got)
	}
}
