package domain

type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Hash    string `json:"-"`
}

func (m Manifest) ReleaseID() string {
	if m.Hash == "" {
		return m.Version
	}
	return m.Version + "@sha256:" + m.Hash
}

type RaceDefinition struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	BaseModifiers map[string]float64 `json:"base_modifiers"`
}

type ItemKind string

const (
	ItemKindMaterial   ItemKind = "material"
	ItemKindComponent  ItemKind = "component"
	ItemKindCurrency   ItemKind = "currency"
	ItemKindKey        ItemKind = "key"
	ItemKindReagent    ItemKind = "reagent"
	ItemKindBlueprint  ItemKind = "blueprint"
	ItemKindConsumable ItemKind = "consumable"
	ItemKindEquipment  ItemKind = "equipment"
	ItemKindTool       ItemKind = "tool"
)

type ItemDefinition struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Kind ItemKind `json:"kind"`
}

type LootEntry struct {
	ItemID      string  `json:"item_id"`
	Probability float64 `json:"probability"`
	MinQuantity int     `json:"min_quantity"`
	MaxQuantity int     `json:"max_quantity"`
}

type LootTableDefinition struct {
	ID      string      `json:"id"`
	Entries []LootEntry `json:"entries"`
}

type MonsterDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Level       int    `json:"level"`
	LootTableID string `json:"loot_table_id"`
}

type RecipeInput struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type RecipeDefinition struct {
	ID        string        `json:"id"`
	Station   string        `json:"station"`
	Inputs    []RecipeInput `json:"inputs"`
	OutputID  string        `json:"output_item_id"`
	OutputQty int           `json:"output_quantity"`
}

type WeightedMonster struct {
	MonsterID string `json:"monster_id"`
	Weight    int    `json:"weight"`
}

type PartyScaling struct {
	MonsterHealthPerPlayer float64 `json:"monster_health_per_player"`
	MonsterDamagePerPlayer float64 `json:"monster_damage_per_player"`
	ExperienceMultiplier   float64 `json:"experience_multiplier"`
	LootMultiplier         float64 `json:"loot_multiplier"`
}

type DungeonTierDefinition struct {
	Tier                     int               `json:"tier"`
	EncounterIntervalSeconds int               `json:"encounter_interval_seconds"`
	MonsterPool              []WeightedMonster `json:"monster_pool"`
	BossID                   string            `json:"boss_id"`
	BossKeyItemID            string            `json:"boss_key_item_id"`
}

type DungeonDefinition struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	PartyScaling PartyScaling            `json:"party_scaling"`
	Tiers        []DungeonTierDefinition `json:"tiers"`
}

type GatheringDrop struct {
	ItemID      string  `json:"item_id"`
	Probability float64 `json:"probability"`
	MinQuantity int     `json:"min_quantity"`
	MaxQuantity int     `json:"max_quantity"`
}

type GatheringDefinition struct {
	ID              string          `json:"id"`
	Discipline      string          `json:"discipline"`
	Name            string          `json:"name"`
	IntervalSeconds int             `json:"interval_seconds"`
	Drops           []GatheringDrop `json:"drops"`
}
