// Command eventcatalog regenerates docs/events.md from domain/events. Run
// from backend/: go run ./cmd/eventcatalog
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/adirkuhn/commerceos/internal/eventcatalog"
)

func main() {
	entries, err := eventcatalog.Collect(filepath.Join("domain", "events"))
	if err != nil {
		log.Fatalf("eventcatalog: %v", err)
	}
	out := filepath.Join("..", "docs", "events.md")
	if err := os.WriteFile(out, []byte(eventcatalog.Markdown(entries)), 0o644); err != nil {
		log.Fatalf("eventcatalog: write %s: %v", out, err)
	}
	log.Printf("eventcatalog: wrote %d events to %s", len(entries), out)
}
