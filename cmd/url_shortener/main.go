package main

import (
	"log"
	"log/slog"

	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	log := logger.LoadLogger(cfg.Env)

	log.Info("starting url_shortener on", slog.String("env", cfg.Env))


	
}