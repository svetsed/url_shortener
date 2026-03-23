package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

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
	counter := 3 // may configure
	shortURL := ""
	var err error
	for counter > 0 {
		shortURL, err = generateRandomString(8)
		if err != nil {
			return nil, err
		}

		// check if this shortURL already exist or not
		_, err = s.repo.GetByShortURL(shortURL)
		if err != nil && errors.Is(err, storage.ErrorNotFound) {
			break
		}
		counter--
		shortURL = ""
	}

	if shortURL == "" {
		return nil, fmt.Errorf("failed to create a unique short URL")
	}

	url := &model.URL{
		ID: "1", // TODO
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

// IsValidURL checks for an empty value, tries to parse the URL struct
// and also checks that such a URL has not previosly been saved to the database.
func (s *Service) IsValidURL(someURL string) bool {
	if someURL == "" {
		return false
	}

	u, err := url.ParseRequestURI(someURL)
	if err != nil {
		return false
	}

	if !u.IsAbs() {
		return false
	}
	
	_, err = s.repo.GetByOringURL(someURL)
	if err != nil && errors.Is(err, storage.ErrorNotFound) {
		return true
	}

	return false
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}