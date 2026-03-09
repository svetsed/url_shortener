package inmemory

import (
	"fmt"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

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

	return nil, storage.ErrorNotFound
}

func (ms *MemoryStorage) GetByOringURL(origURL string) (*model.URL, error) {
	for _, url := range ms.urls {
		if url.OriginalURL == origURL {
			return &url, nil
		}
	}

	return nil, storage.ErrorNotFound
}