package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/svetsed/url_shortener/internal/config"
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

	if string(origURL) == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !h.service.IsValidURL(string(origURL)) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// InternalServerError?
	shortURL, err := h.service.CreateShortURL(string(origURL))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// принятие короткого url если есть или создание нового короткого url
	// проверка на уникальность короткого url (отдельная функция)

	url := h.cfg.BaseAddress + shortURL.ShortURL
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(url)))
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(url))
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
	http.Redirect(w, r, foundOrigURL, http.StatusTemporaryRedirect)
}