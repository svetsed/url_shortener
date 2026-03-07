package storage

import (
	"errors"
	"fmt"

	"github.com/svetsed/url_shortener/internal/model"
)

// var Storage = []model.URL{}

var (
	ErrorNotFound = errors.New("not found")
)

type Repository interface {
	Save(url *model.URL) error
	GetByShortURL(shortURL string) (*model.URL, error)
}

type MemoryStorage struct {
	urls []model.URL
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make([]model.URL, 0),
	}
}

func (ms *MemoryStorage) Save(url *model.URL) error {
	if ms.urls != nil {
		ms.urls = append(ms.urls, *url)
		return nil
	}
	return fmt.Errorf("storage not initialized")
}

func (ms *MemoryStorage) GetByShortURL(shortURL string) (*model.URL, error) {
	for _, url := range ms.urls {
		if url.ShortURL == shortURL {
			return &url, nil
		}
	}

	return nil, ErrorNotFound
}