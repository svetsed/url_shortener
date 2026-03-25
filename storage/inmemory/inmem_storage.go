package inmemory

import (
	"sync"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

type memoryStorage struct {
	urls []model.URL
	mu	 sync.RWMutex
}

func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{
		urls: make([]model.URL, 0),
	}
}

func (ms *memoryStorage) Save(url *model.URL) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.urls != nil {
		ms.urls = append(ms.urls, *url)
		return nil
	}
	return storage.ErrorStorageNotInitialized
}

func (ms *memoryStorage) GetByShortURL(shortURL string) (*model.URL, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	for _, url := range ms.urls {
		if url.ShortURL == shortURL {
			return &url, nil
		}
	}

	return nil, storage.ErrorNotFound
}

func (ms *memoryStorage) GetByOringURL(origURL string) (*model.URL, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	for _, url := range ms.urls {
		if url.OriginalURL == origURL {
			return &url, nil
		}
	}

	return nil, storage.ErrorNotFound
}

func (ms *memoryStorage) Close() error {
	return nil
}