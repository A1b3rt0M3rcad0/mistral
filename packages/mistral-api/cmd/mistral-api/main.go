package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/httpapi"
	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	contentcomposition "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/composition"
	contententrypoint "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/entrypoint"
	craftingapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
)

func main() {
	contentRoot := flag.String("content", "./content", "path to game content root")
	listen := flag.String("listen", ":8080", "HTTP listen address")
	databaseURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL; persistence is disabled when empty")
	migrate := flag.Bool("migrate", false, "apply pending SQL migrations before serving")
	migrationRoot := flag.String("migrations", "./migrations", "path to SQL migrations")
	flag.Parse()

	contentService := contentcomposition.NewService(*contentRoot)
	registry, err := contententrypoint.New(contentService).LoadRelease()
	if err != nil {
		log.Fatalf("load content release: %v", err)
	}

	options := []httpapi.Option{}
	if *databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		db, err := database.OpenPostgres(ctx, *databaseURL)
		if err != nil {
			cancel()
			log.Fatal(err)
		}
		if *migrate {
			if err := dbmigrate.Apply(ctx, db, *migrationRoot); err != nil {
				cancel()
				_ = db.Close()
				log.Fatal(err)
			}
		}
		cancel()
		defer func() { _ = db.Close() }()
		gameplay, err := gameplaydb.New(db)
		if err != nil {
			log.Fatal(err)
		}
		registration := characterapplication.NewPersistedRegistrationService(
			characterapplication.NewService(registry),
			gameplay.Characters,
			gameplay.Inventories,
			gameplay.Ownership,
			gameplay.Transactor,
			gameplay.Idempotency,
		)
		crafting := craftingapplication.NewPersistedCraftService(
			craftingapplication.NewService(registry),
			gameplay.Inventories,
			gameplay.Transactor,
			gameplay.Idempotency,
		)
		options = append(options,
			httpapi.WithGameplayStore(gameplay),
			httpapi.WithCharacterRegistrar(registration),
			httpapi.WithCharacterCrafter(crafting),
			httpapi.WithCharacterAuthorizer(identityapplication.NewAuthorizer(gameplay.Ownership)),
		)
	}

	server := httpapi.New(registry, options...)
	fmt.Printf("mistral-api listening on %s with content %s@%s\n", *listen, registry.Manifest.Name, registry.Manifest.ReleaseID())
	if err := http.ListenAndServe(*listen, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
