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
	"github.com/svetsed/url_shortener/storage"
	filestorage "github.com/svetsed/url_shortener/storage/file_storage"
	"github.com/svetsed/url_shortener/storage/inmemory"
	"github.com/svetsed/url_shortener/storage/postgres"
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
		sugarLog.Fatalf("config initialization error: %v", err)
	}

	var repo storage.Repository
	var closeFunc func() error

	switch {
	case cfg.DatabaseDSN != "":
		pg, err := postgres.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			sugarLog.Fatalf("postgres storage initialization error: %v", err)
		}
		repo = pg
		closeFunc = pg.Close

		sugarLog.Info("storage in the postgres database is selected")
	case cfg.FileStoragePath != "":
		fs, err := filestorage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			sugarLog.Fatalf("file storage initialization error: %v", err)
		}

		repo = fs
		closeFunc = fs.Close

		sugarLog.Infof("file storage selected: %s", cfg.FileStoragePath)
	default:
		repo = inmemory.NewMemoryStorage()
		closeFunc = nil

		sugarLog.Info("memory storage selected")
	}

	if closeFunc != nil {
		defer func() {
			if err := closeFunc(); err != nil {
				sugarLog.Errorf("close error: %v", err)
			}
		}()
	}

	serv := service.NewService(repo)
	h := handler.NewHandler(serv, cfg, sugarLog)

	r := chi.NewRouter()

	r.Use(middleware.Recoverer,
		middleware.RequestID,
		logger.LoggingMiddleware(sugarLog),
		compress.GzipMiddleware,
	)

	r.Post("/", h.CreateShortURLHandler)
	r.Post("/api/shorten", h.CreateShortURLHandlerFromJSON)
	r.Get("/{id}", h.RedirectToOrigURLHandler)
	r.Get("/ping", h.HealthCheckDBHandler)

	sugarLog.Infof("Server starts with: server address - %s, base url - %s", cfg.LoadAddress, cfg.BaseAddress)

	sugarLog.Fatal(http.ListenAndServe(cfg.LoadAddress, r))
}