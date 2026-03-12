package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/svetsed/url_shortener/internal/server/handler"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/storage/inmemory"
)

func main() {
	repo := inmemory.NewMemoryStorage()
	serv := service.NewService(repo)
	h := handler.NewHandler(serv)

	r := chi.NewRouter()

	r.Post("/", h.CreateShortURLHandler)
	r.Get("/{id}", h.RedirectToOrigURLHandler)

	log.Fatal(http.ListenAndServe(":8080", r))
}