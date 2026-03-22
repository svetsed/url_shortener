package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/server/handler"
	"github.com/svetsed/url_shortener/internal/server/own_middleware/compress"
	"github.com/svetsed/url_shortener/internal/server/own_middleware/logger"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/storage/inmemory"
	"go.uber.org/zap"
)

func main() {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("logger initialization error: %v", err)
	}

	sugarLog := zapLogger.Sugar()

	cfg := config.NewDefaultConfig()
	if err := config.SettingConfig(cfg); err != nil {
		sugarLog.Errorf("config initialization error: %v", err)
	}

	repo := inmemory.NewMemoryStorage()
	serv := service.NewService(repo)
	h := handler.NewHandler(serv, cfg)

	r := chi.NewRouter()

	r.Use(middleware.Recoverer,
		middleware.RequestID,
		logger.LoggingMiddleware(sugarLog),
		compress.GzipMiddleware,
	)

	r.Post("/", h.CreateShortURLHandler)
	r.Post("/api/shorten", h.CreateShortURLHandlerFromJSON)
	r.Get("/{id}", h.RedirectToOrigURLHandler)

	sugarLog.Infof("Server starts with: server address - %s, base url - %s\n", cfg.LoadAddress, cfg.BaseAddress)

	sugarLog.Fatal(http.ListenAndServe(cfg.LoadAddress, r))
}