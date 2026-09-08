package domain

import "time"

type Session struct {
	ID             string    `json:"id"`
	CharacterID    string    `json:"character_id"`
	ContentRelease string    `json:"content_release"`
	GatheringID    string    `json:"gathering_id"`
	Seed           int64     `json:"seed"`
	StartedAt      time.Time `json:"started_at"`
	ClaimedCycles  int       `json:"claimed_cycles"`
}

type Reward struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type RewardBatch struct {
	Cycle      int       `json:"cycle"`
	AcquiredAt time.Time `json:"acquired_at"`
	ItemID     string    `json:"item_id"`
	Quantity   int       `json:"quantity"`
}

type Resolution struct {
	SessionID    string        `json:"session_id"`
	ResolvedAt   time.Time     `json:"resolved_at"`
	FromCycle    int           `json:"from_cycle"`
	ThroughCycle int           `json:"through_cycle"`
	Rewards      []Reward      `json:"rewards"`
	Batches      []RewardBatch `json:"batches"`
}
