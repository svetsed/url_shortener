package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/svetsed/url_shortener/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
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
		fmt.Println(err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// InternalServerError?
	shortURL, err := h.service.CreateShortURL(string(origURL))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// проверка на валидный url
	// проверка что такой url уже есть в базе или нет
	// принятие короткого url если есть или создание нового короткого url
	// проверка на уникальность короткого url (отдельная функция)
	// сохранение 2 url в базу данных
	// вернуть ответ пользователю с новой ссылкой

	url := "http://localhost:8080/" + shortURL.ShortURL

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(url)))
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(url)) // short url
}

func (h *Handler) RedirectToOrigURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortURL := r.PathValue("id")
	// проверка что такой короткий url существует (отдельная функция)?
	// получение оригинального url из базы
	// редирект на него с 307
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

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", foundOrigURL) // orig url
	w.WriteHeader(http.StatusTemporaryRedirect)

	http.Redirect(w, r, foundOrigURL, http.StatusTemporaryRedirect)
}