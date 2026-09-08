package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/httpapi"
	contentcomposition "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/composition"
	contententrypoint "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/entrypoint"
)

func main() {
	contentRoot := flag.String("content", "./content", "path to game content root")
	listen := flag.String("listen", ":8080", "HTTP listen address")
	flag.Parse()

	contentService := contentcomposition.NewService(*contentRoot)
	registry, err := contententrypoint.New(contentService).LoadRelease()
	if err != nil {
		log.Fatalf("load content release: %v", err)
	}

	server := httpapi.New(registry)
	fmt.Printf("mistral-api listening on %s with content %s@%s\n", *listen, registry.Manifest.Name, registry.Manifest.ReleaseID())
	if err := http.ListenAndServe(*listen, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
