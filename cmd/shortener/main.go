package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/server/handler"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/storage/inmemory"
)

func main() {
	cfg := config.Config{}
	if err := config.SettingConfig(&cfg); err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	repo := inmemory.NewMemoryStorage()
	serv := service.NewService(repo)
	h := handler.NewHandler(serv, &cfg)

	r := chi.NewRouter()

	r.Post("/", h.CreateShortURLHandler)
	r.Get("/{id}", h.RedirectToOrigURLHandler)

	log.Fatal(http.ListenAndServe(cfg.LoadAddress, r))
}