package domain

import "time"

type CharacterSnapshot struct {
	CharacterID string `json:"character_id"`
	Level       int    `json:"level"`
	Attack      int    `json:"attack"`
	Defense     int    `json:"defense"`
	MaxHealth   int    `json:"max_health"`
}

type Run struct {
	ID             string            `json:"id"`
	ContentRelease string            `json:"content_release"`
	DungeonID      string            `json:"dungeon_id"`
	DungeonTier    int               `json:"dungeon_tier"`
	PartySize      int               `json:"party_size"`
	Seed           int64             `json:"seed"`
	StartedAt      time.Time         `json:"started_at"`
	Character      CharacterSnapshot `json:"character_snapshot"`
}

type Encounter struct {
	Ordinal   int       `json:"ordinal"`
	DueAt     time.Time `json:"due_at"`
	MonsterID string    `json:"monster_id"`
}

type Resolution struct {
	RunID      string      `json:"run_id"`
	ResolvedAt time.Time   `json:"resolved_at"`
	Encounters []Encounter `json:"encounters"`
}
