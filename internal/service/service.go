package service

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

type Service struct {
	repo storage.Repository
}

func NewService(repo storage.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateShortURL(origURL string) (*model.URL, error) {
	shortURL, err := generateRandomString(8)
	if err != nil {
		return nil, err
	}

	url := &model.URL{
		ID: "1", // 
		ShortURL: shortURL,
		OriginalURL: origURL,
	}

	err = s.repo.Save(url)
	if err != nil {
		return nil, err
	}

	return url, nil
}

func (s *Service) GetOriginalURL(shortURL string) (string, error) {
	url, err := s.repo.GetByShortURL(shortURL)
	if err != nil {
		return "", err
	}

	return url.OriginalURL, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}