package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/infra/jsonloader"
)

func main() {
	contentRoot := flag.String("content", "./content", "path to game content root")
	flag.Parse()

	service := application.NewService(jsonloader.New(*contentRoot))
	registry, err := service.LoadValidatedRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "content validation failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"content %s@%s valid: %d races, %d items, %d monsters, %d dungeons, %d recipes, %d gathering areas\n",
		registry.Manifest.Name,
		registry.Manifest.ReleaseID(),
		len(registry.Races),
		len(registry.Items),
		len(registry.Monsters),
		len(registry.Dungeons),
		len(registry.Recipes),
		len(registry.Gathering),
	)
}
