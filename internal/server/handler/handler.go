package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	cfg     *config.Config
	sugarLog *zap.SugaredLogger
}

func NewHandler(service *service.Service, cfg *config.Config, sugarLog *zap.SugaredLogger) *Handler {
	return &Handler{
		service: service,
		cfg: cfg,
		sugarLog: sugarLog,
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

	url, err := h.service.CreateShortURL(string(origURL))
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	err = h.service.SaveOneURL(url)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}


	urlStr := h.cfg.BaseAddress + "/" + url.ShortURL
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(urlStr)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(urlStr))
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

	var reqURL model.OneURLRequest
	if err := json.Unmarshal(data, &reqURL); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !h.service.IsValidURL(string(reqURL.URL)) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	url, err := h.service.CreateShortURL(string(reqURL.URL))
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	err = h.service.SaveOneURL(url)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	res := model.OneURLResponse{
		Result: h.cfg.BaseAddress + "/" + url.ShortURL,
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

func (h *Handler) CreateShortURLsBatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	defer r.Body.Close()
	reqURLs := []model.ManyURLRequest{}
	err := json.NewDecoder(r.Body).Decode(&reqURLs)
	if err != nil {
		h.sugarLog.Errorf("json-error when reading from body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(reqURLs) == 0 {
		h.sugarLog.Error("length of struct reqURLs = 0")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	respURLs := []model.ManyURLResponse{}
	urlsForSave := make([]*model.URL, 0)
	for _, reqURL := range reqURLs {
		if !h.service.IsValidURL(string(reqURL.OriginalURL)) {
			mes := fmt.Sprintf("bad request: url with correlation_id = %s is not valid (url = %s)", reqURL.CorrelationID, reqURL.OriginalURL)
			h.sugarLog.Error(mes)
			http.Error(w, mes, http.StatusBadRequest)
			return
		}

		url, err := h.service.CreateShortURL(string(reqURL.OriginalURL))
		if err != nil {
			h.sugarLog.Errorf("failed to create short url: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		urlsForSave = append(urlsForSave, url)

		respURLs = append(respURLs, model.ManyURLResponse{
			CorrelationID: reqURL.CorrelationID,
			ShortURL: h.cfg.BaseAddress + "/" + url.ShortURL,
		})
	}

	err = h.service.SaveManyURL(urlsForSave)
	if err != nil {
		h.sugarLog.Errorf("failed to save many urls: %w", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	respData, err := json.Marshal(&respURLs)
	if err != nil {
		h.sugarLog.Errorf("json-error from Marshal: %v", err)
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

func (h *Handler) HealthCheckDBHandler(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		h.sugarLog.Error("service not initialized")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.Ping(ctx); err != nil {
		h.sugarLog.Errorw("database ping failed", "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

