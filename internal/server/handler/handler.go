package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/service"
)

type Handler struct {
	service *service.Service
	cfg     *config.Config
}

func NewHandler(service *service.Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg: cfg,
	}
}

func (h *Handler) CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	defer r.Body.Close()
	origURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !h.service.IsValidURL(string(origURL)) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.CreateShortURL(string(origURL))
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	url := h.cfg.BaseAddress + "/" + shortURL.ShortURL
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(url)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(url))
}

func (h *Handler) CreateShortURLHandlerFromJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(data) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var reqURL model.RequestJSON
	if err := json.Unmarshal(data, &reqURL); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !h.service.IsValidURL(string(reqURL.URL)) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.CreateShortURL(string(reqURL.URL))
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	res := model.ResponseJSON{
		Result: h.cfg.BaseAddress + "/" + shortURL.ShortURL,
	}

	respData, err := json.Marshal(&res)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respData)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	w.WriteHeader(http.StatusCreated)

	w.Write(respData)
}

func (h *Handler) RedirectToOrigURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortURL := chi.URLParam(r, "id")
	if shortURL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// not_found?
	foundOrigURL, err := h.service.GetOriginalURL(shortURL)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(foundOrigURL)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	http.Redirect(w, r, foundOrigURL, http.StatusTemporaryRedirect)
}