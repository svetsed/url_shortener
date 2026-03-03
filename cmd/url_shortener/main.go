package main

import (
	"log"
	"net/http"

	"github.com/svetsed/url_shortener/internal/server/handler"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler.CreateShortURLHandler)
	mux.HandleFunc("/{id}", handler.RedirectToOrigURLHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}