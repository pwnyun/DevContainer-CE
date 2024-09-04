package main

import (
	"log"

	"github.com/AethoceSora/DevContainer/src/internal/api"
	"github.com/AethoceSora/DevContainer/src/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	api.StartServer(cfg)
}
