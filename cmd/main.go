package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wltbagent/quicksend/handler"
	"github.com/wltbagent/quicksend/storage"
)

func main() {
	addr := os.Getenv("QUICKSEND_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dataDir := os.Getenv("QUICKSEND_DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	store, err := storage.New(dataDir)
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}

	go janitor(store)

	h := handler.New(store)
	log.Printf("quicksend listening on %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func janitor(store *storage.Storage) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		removed := store.PurgeExpired(24 * time.Hour)
		if removed > 0 {
			log.Printf("janitor: removed %d expired files", removed)
		}
	}
}
