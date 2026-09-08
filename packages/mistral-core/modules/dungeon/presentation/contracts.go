package presentation

import "time"

type PreviewRequest struct {
	RunID          string    `json:"run_id"`
	DungeonID      string    `json:"dungeon_id"`
	DungeonTier    int       `json:"dungeon_tier"`
	PartySize      int       `json:"party_size"`
	Seed           int64     `json:"seed"`
	StartedAt      time.Time `json:"started_at"`
	CharacterID    string    `json:"character_id"`
	CharacterLevel int       `json:"character_level"`
}
