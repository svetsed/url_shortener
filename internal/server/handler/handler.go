package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

func CreateRandomString(len int) (string, error) {
	len = 8
	bytes := make([]byte, len)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:len], nil
}

func CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// а не 500 >?
	shortUrl, err := CreateRandomString(8)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tmp := model.URL{
		OriginalURL: string(data),
		ShortURL: shortUrl,
	}

	storage.Storage = append(storage.Storage, tmp)

	// проверка на валидный url
	// проверка что такой url уже есть в базе или нет
	// принятие короткого url если есть или создание нового короткого url
	// проверка на уникальность короткого url (отдельная функция)
	// сохранение 2 url в базу данных
	// вернуть ответ пользователю с новой ссылкой

	url := "http://localhost:8080/" + shortUrl

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

	foundOrigUrl := ""
	for _, url := range storage.Storage {
		if url.ShortURL == id {
			foundOrigUrl = url.OriginalURL
			break
		}
	}

	// not_found?
	if foundOrigUrl == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", foundOrigUrl) // orig url
	w.WriteHeader(http.StatusTemporaryRedirect)
}