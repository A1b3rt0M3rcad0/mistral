package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/httpapi"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/httphost"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/runtimeid"
	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	contentcomposition "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/composition"
	contententrypoint "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/entrypoint"
	craftingapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
)

func main() {
	poolDefaults := database.DefaultPoolConfig()
	contentRoot := flag.String("content", "./content", "path to game content root")
	listen := flag.String("listen", ":8080", "HTTP listen address")
	databaseURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL; persistence is disabled when empty")
	databaseMaxOpenConns := flag.Int("database-max-open-conns", poolDefaults.MaxOpenConns, "maximum open PostgreSQL connections")
	databaseMaxIdleConns := flag.Int("database-max-idle-conns", poolDefaults.MaxIdleConns, "maximum idle PostgreSQL connections")
	databaseConnMaxLifetime := flag.Duration("database-conn-max-lifetime", poolDefaults.ConnMaxLifetime, "maximum PostgreSQL connection lifetime")
	databaseConnMaxIdleTime := flag.Duration("database-conn-max-idle-time", poolDefaults.ConnMaxIdleTime, "maximum PostgreSQL connection idle time")
	migrate := flag.Bool("migrate", false, "apply pending SQL migrations before serving")
	migrationRoot := flag.String("migrations", "./migrations", "path to SQL migrations")
	readHeaderTimeout := flag.Duration("read-header-timeout", 5*time.Second, "maximum time to read HTTP request headers")
	readTimeout := flag.Duration("read-timeout", 15*time.Second, "maximum time to read a complete HTTP request")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "maximum time to write an HTTP response")
	idleTimeout := flag.Duration("idle-timeout", 60*time.Second, "maximum keep-alive idle time")
	shutdownTimeout := flag.Duration("shutdown-timeout", 10*time.Second, "graceful HTTP shutdown timeout")
	maxHeaderBytes := flag.Int("max-header-bytes", 64<<10, "maximum total size of HTTP request headers")
	flag.Parse()

	if *readHeaderTimeout <= 0 || *readTimeout <= 0 || *writeTimeout <= 0 || *idleTimeout <= 0 || *shutdownTimeout <= 0 || *maxHeaderBytes <= 0 {
		log.Fatal("HTTP timeouts and max-header-bytes must be positive")
	}

	contentService := contentcomposition.NewService(*contentRoot)
	registry, err := contententrypoint.New(contentService).LoadRelease()
	if err != nil {
		log.Fatalf("load content release: %v", err)
	}

	options := []httpapi.Option{}
	if *databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		db, err := database.OpenPostgresWithPool(ctx, *databaseURL, database.PoolConfig{
			MaxOpenConns:    *databaseMaxOpenConns,
			MaxIdleConns:    *databaseMaxIdleConns,
			ConnMaxLifetime: *databaseConnMaxLifetime,
			ConnMaxIdleTime: *databaseConnMaxIdleTime,
		})
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
			runtimeid.Generator{},
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

	api := httpapi.New(registry, options...)
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: *readHeaderTimeout,
		ReadTimeout:       *readTimeout,
		WriteTimeout:      *writeTimeout,
		IdleTimeout:       *idleTimeout,
		MaxHeaderBytes:    *maxHeaderBytes,
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("mistral-api listening on %s with content %s@%s\n", listener.Addr().String(), registry.Manifest.Name, registry.Manifest.ReleaseID())
	if err := httphost.Run(runCtx, httpServer, listener, *shutdownTimeout); err != nil {
		log.Fatal(err)
	}
}
