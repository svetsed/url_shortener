package inmemory

import (
	"sync"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

var _ storage.Repository = (*memoryStorage)(nil)

type memoryStorage struct {
	urls []model.URL
	mu	 sync.RWMutex
}

func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{
		urls: make([]model.URL, 0),
	}
}


// ---------------- Implement Repository ----------------

func (ms *memoryStorage) Save(url *model.URL) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.urls == nil {
		return storage.ErrorStorageNotInitialized
	}

	ms.urls = append(ms.urls, *url)
	return nil
}

func (ms *memoryStorage) SaveManyURL(newURLs []*model.URL) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.urls == nil {
		return storage.ErrorStorageNotInitialized
	}

	for _, url := range newURLs {
		ms.urls = append(ms.urls, *url)
	}

	return nil
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