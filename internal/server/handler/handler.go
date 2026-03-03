package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/storage"
)

func CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
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

	// а не 500 >?
	shortURL, err := service.CreateRandomString(8)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tmp := model.URL{
		OriginalURL: string(data),
		ShortURL: shortURL,
	}

	storage.Storage = append(storage.Storage, tmp)

	// проверка на валидный url
	// проверка что такой url уже есть в базе или нет
	// принятие короткого url если есть или создание нового короткого url
	// проверка на уникальность короткого url (отдельная функция)
	// сохранение 2 url в базу данных
	// вернуть ответ пользователю с новой ссылкой

	url := "http://localhost:8080/" + shortURL

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(url)))
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(url)) // short url

}

func RedirectToOrigURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	// проверка что такой короткий url существует (отдельная функция)
	// получение оригинального url из базы
	// редирект на него с 307
	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	foundOrigURL := ""
	for _, url := range storage.Storage {
		if url.ShortURL == id {
			foundOrigURL = url.OriginalURL
			break
		}
	}

	// not_found?
	if foundOrigURL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", foundOrigURL) // orig url
	w.WriteHeader(http.StatusTemporaryRedirect)
}