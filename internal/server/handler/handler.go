package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/server/auth"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/internal/storage"
	"go.uber.org/zap"
)

type Handler struct {
	service  *service.Service
	cfg      *config.Config
	sugarLog *zap.SugaredLogger
}

func NewHandler(service *service.Service, cfg *config.Config, sugarLog *zap.SugaredLogger) *Handler {
	return &Handler{
		service:  service,
		cfg:      cfg,
		sugarLog: sugarLog,
	}
}

// Post /
func (h *Handler) CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetOrCreateUserID(w, r)
	if err != nil {
		h.sugarLog.Errorf("error from auth.GetOrCreateUserID(): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()
	origURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var url *model.URL
	status := http.StatusCreated
	skipCreating := false
	urlExist, err := h.service.IsValidURL(string(origURL), userID)
	if err != nil {
		if errors.Is(err, storage.ErrURLAlreadyExist) {
			url = urlExist
			status = http.StatusConflict
			skipCreating = true
		} else if !errors.Is(err, storage.ErrorNotFound) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}

	if !skipCreating {
		newURL, err := h.service.CreateShortURL(string(origURL))
		if err != nil {
			h.sugarLog.Errorf("error from service.CreateShortURL(): %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		newURL.UserID = userID

		err = h.service.SaveOneURL(newURL)
		if err != nil {
			h.sugarLog.Errorf("error from service.SaveOneURL(): %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		url = newURL
	}

	urlStr := h.cfg.BaseAddress + "/" + url.ShortURL
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(urlStr)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	w.WriteHeader(status)

	w.Write([]byte(urlStr))
}

// Post /api/shorten
func (h *Handler) CreateShortURLHandlerFromJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetOrCreateUserID(w, r)
	if err != nil {
		h.sugarLog.Errorf("error from auth.GetOrCreateUserID(): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
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

	var url *model.URL
	status := http.StatusCreated
	skipCreating := false
	urlExist, err := h.service.IsValidURL(string(reqURL.URL), userID)
	if err != nil {
		if errors.Is(err, storage.ErrURLAlreadyExist) {
			url = urlExist
			status = http.StatusConflict
			skipCreating = true
		} else if !errors.Is(err, storage.ErrorNotFound) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}

	if !skipCreating {
		newURL, err := h.service.CreateShortURL(string(reqURL.URL))
		if err != nil {
			h.sugarLog.Errorf("error from service.CreateShortURL(): %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		newURL.UserID = userID

		err = h.service.SaveOneURL(newURL)
		if err != nil {
			h.sugarLog.Errorf("error from service.SaveOneURL(): %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		url = newURL
	}

	res := model.OneURLResponse{
		Result: h.cfg.BaseAddress + "/" + url.ShortURL,
	}

	respData, err := json.Marshal(&res)
	if err != nil {
		h.sugarLog.Errorf("error from json.Marshal(&OneURLResponse): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respData)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	w.WriteHeader(status)

	w.Write(respData)
}

// Post /api/shorten/batch
func (h *Handler) CreateShortURLsBatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetOrCreateUserID(w, r)
	if err != nil {
		h.sugarLog.Errorf("error from auth.GetOrCreateUserID(): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()
	reqURLs := []model.ManyURLRequest{}
	err = json.NewDecoder(r.Body).Decode(&reqURLs)
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

	respURLs := make([]model.ManyURLResponse, 0, len(reqURLs))
	urlsForSave := make([]*model.URL, 0)
	skipCreating := false
	for _, reqURL := range reqURLs {
		skipCreating = false
		var url model.URL
		urlExist, err := h.service.IsValidURL(string(reqURL.OriginalURL), userID)
		if err != nil {
			if errors.Is(err, storage.ErrURLAlreadyExist) {
				skipCreating = true
				url = *urlExist
			} else if !errors.Is(err, storage.ErrorNotFound) {
				mes := fmt.Sprintf("bad request: url with correlation_id = %s is not valid (url = %s)", reqURL.CorrelationID, reqURL.OriginalURL)
				h.sugarLog.Error(mes)
				http.Error(w, mes, http.StatusBadRequest)
				return
			}

			h.sugarLog.Info(err)
		}

		if !skipCreating {
			newURL, err := h.service.CreateShortURL(string(reqURL.OriginalURL))
			if err != nil {
				h.sugarLog.Errorf("failed to create short url: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}

			url = *newURL
			url.UserID = userID

			urlsForSave = append(urlsForSave, &url)
		}

		respURLs = append(respURLs, model.ManyURLResponse{
			CorrelationID: reqURL.CorrelationID,
			ShortURL:      h.cfg.BaseAddress + "/" + url.ShortURL,
		})
	}

	err = h.service.SaveManyURL(urlsForSave)
	if err != nil {
		h.sugarLog.Errorf("failed to save many urls: %v", err)
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

// Get /api/user/urls
func (h *Handler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetOrCreateUserID(w, r)
	if err != nil {
		h.sugarLog.Errorf("error from auth.GetOrCreateUserID(): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	userURLs, err := h.service.GetUserURLs(userID)
	if err != nil {
		h.sugarLog.Errorf("error from service.GetUserURLs(userID): %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	if len(userURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := []map[string]string{}
	for _, url := range userURLs {
		if url.NeedDelete {
			continue
		}
		response = append(response, map[string]string{
			"short_url":    h.cfg.BaseAddress + "/" + url.ShortURL,
			"original_url": url.OriginalURL,
		})
	}

	if len(response) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Get /{id}
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

	if foundOrigURL.NeedDelete {
		http.Error(w, "url has been removed", http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(foundOrigURL.OriginalURL)))
	w.Header().Set("X-Request-ID", middleware.GetReqID(r.Context()))
	http.Redirect(w, r, foundOrigURL.OriginalURL, http.StatusTemporaryRedirect)
}

// Get /ping
func (h *Handler) HealthCheckDBHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

// Delete /api/user/urls
func (h *Handler) DeleteUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		auth.CreateNewUser(w)
		h.sugarLog.Errorf("error from auth.GetUserIDFromCookie(r): %v", err)
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	shortURLs := []string{}
	err = json.NewDecoder(r.Body).Decode(&shortURLs)
	if err != nil {
		h.sugarLog.Errorf("json-error when reading from body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(shortURLs) == 0 {
		h.sugarLog.Error("length of slice with shortURLs = 0")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	go func() {
		err := h.service.MarkAsDeleted(shortURLs, userID)
		if err != nil {
			h.sugarLog.Errorf("failed to mark URLs as deleted for user %s: %v", userID, err)
		} else {
			h.sugarLog.Infof("successfully marked %d URLs as deleted for user %s", len(shortURLs), userID)
		}
	}()

	w.WriteHeader(http.StatusAccepted)

	// когда-то удаляем
}
