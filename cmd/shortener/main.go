package main

import (
	"log"
	"net/http"

	"github.com/svetsed/url_shortener/internal/server/handler"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/storage/inmemory"
)

func main() {
	repo := inmemory.NewMemoryStorage()
	serv := service.NewService(repo)
	h := handler.NewHandler(serv)

	mux := http.NewServeMux()

	mux.HandleFunc("/", h.CreateShortURLHandler)
	mux.HandleFunc("/{id}", h.RedirectToOrigURLHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}