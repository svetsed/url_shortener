package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/server/handler"
	"github.com/svetsed/url_shortener/internal/server/own_middleware/compress"
	"github.com/svetsed/url_shortener/internal/server/own_middleware/logger"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/internal/storage"
	filestorage "github.com/svetsed/url_shortener/internal/storage/file_storage"
	"github.com/svetsed/url_shortener/internal/storage/inmemory"
	"github.com/svetsed/url_shortener/internal/storage/postgres"
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
		pg, err := postgres.NewPostgresStorage(cfg.DatabaseDSN, sugarLog)
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

	workDir, _ := os.Getwd()
	webDir := filepath.Join(workDir, "web")

	// Статика
	fs := http.FileServer(http.Dir(webDir))
	r.Handle("/static/*", http.StripPrefix("/static/", forceMime(fs)))

	// favicon
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "favicon.ico"))
	})

	// Главная страница
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})

	r.Post("/api/shorten", h.CreateShortURLHandlerFromJSON)
	r.Post("/api/shorten/batch", h.CreateShortURLsBatchHandler)
	r.Get("/api/user/urls", h.GetUserURLsHandler)
	r.Delete("/api/user/urls", h.DeleteUserURLsHandler)
	r.Post("/", h.CreateShortURLHandler)
	r.Get("/{id}", h.RedirectToOrigURLHandler)
	r.Get("/ping", h.HealthCheckDBHandler)

	sugarLog.Infof("server starts with: server address - %s, base url - %s", cfg.LoadAddress, cfg.BaseAddress)
	sugarLog.Fatal(http.ListenAndServe(cfg.LoadAddress, r))
}

func forceMime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		next.ServeHTTP(w, r)
	})
}
