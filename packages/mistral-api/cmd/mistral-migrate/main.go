package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
)

func main() {
	databaseURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL")
	migrationRoot := flag.String("migrations", "./migrations", "path to SQL migrations")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.OpenPostgres(ctx, *databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := dbmigrate.Apply(ctx, db, *migrationRoot); err != nil {
		log.Fatal(err)
	}
	log.Println("Mistral database migrations applied")
}
