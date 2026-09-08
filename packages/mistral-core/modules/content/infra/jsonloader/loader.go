package jsonloader

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Loader struct {
	root string
}

func New(root string) Loader {
	return Loader{root: root}
}

func (l Loader) Load() (domain.Registry, error) {
	registry := domain.NewRegistry()
	if err := readJSON(filepath.Join(l.root, "manifest.json"), &registry.Manifest); err != nil {
		return domain.Registry{}, err
	}

	loaders := []func() error{
		func() error {
			return loadDirectory(filepath.Join(l.root, "races"), func(v domain.RaceDefinition) error {
				return addUnique(registry.Races, v.ID, v, "race")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "items"), func(v domain.ItemDefinition) error {
				return addUnique(registry.Items, v.ID, v, "item")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "loot_tables"), func(v domain.LootTableDefinition) error {
				return addUnique(registry.LootTables, v.ID, v, "loot table")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "monsters"), func(v domain.MonsterDefinition) error {
				return addUnique(registry.Monsters, v.ID, v, "monster")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "recipes"), func(v domain.RecipeDefinition) error {
				return addUnique(registry.Recipes, v.ID, v, "recipe")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "dungeons"), func(v domain.DungeonDefinition) error {
				return addUnique(registry.Dungeons, v.ID, v, "dungeon")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "gathering"), func(v domain.GatheringDefinition) error {
				return addUnique(registry.Gathering, v.ID, v, "gathering area")
			})
		},
		func() error {
			return loadDirectory(filepath.Join(l.root, "decay"), func(v domain.DecayDefinition) error {
				return addUnique(registry.Decay, v.ID, v, "decay")
			})
		},
	}
	for _, load := range loaders {
		if err := load(); err != nil {
			return domain.Registry{}, err
		}
	}

	hash, err := contentHash(l.root)
	if err != nil {
		return domain.Registry{}, err
	}
	registry.Manifest.Hash = hash
	return registry, nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values are not allowed", path)
		}
		return fmt.Errorf("decode %s trailing data: %w", path, err)
	}
	return nil
}

func loadDirectory[T any](directory string, add func(T) error) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read content directory %s: %w", directory, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var value T
		if err := readJSON(filepath.Join(directory, entry.Name()), &value); err != nil {
			return err
		}
		if err := add(value); err != nil {
			return fmt.Errorf("load %s: %w", filepath.Join(directory, entry.Name()), err)
		}
	}
	return nil
}

func addUnique[T any](target map[string]T, id string, value T, kind string) error {
	if id == "" {
		return fmt.Errorf("%s id is required", kind)
	}
	if _, exists := target[id]; exists {
		return fmt.Errorf("duplicate %s id %q", kind, id)
	}
	target[id] = value
	return nil
}

func contentHash(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk content for hash: %w", err)
	}
	sort.Strings(files)

	h := sha256.New()
	for _, path := range files {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("hash content file %s: %w", path, err)
		}
		_, _ = h.Write([]byte(filepath.ToSlash(relative)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
