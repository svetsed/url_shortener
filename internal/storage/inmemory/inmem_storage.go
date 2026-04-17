package inmemory

import (
	"fmt"
	"sync"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/storage"
)

var _ storage.Repository = (*memoryStorage)(nil)

type memoryStorage struct {
	urls []model.URL
	mu   sync.RWMutex
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

func (ms *memoryStorage) GetUserURLs(userID string) ([]model.URL, error) {
	userURLs := make([]model.URL, 0)
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, url := range ms.urls {
		if url.UserID == userID {
			userURLs = append(userURLs, url)
		}
	}

	return userURLs, nil
}

func (ms *memoryStorage) MarkAsDeleted(shortURLs []string, userID string) error {
	if shortURLs == nil {
		return fmt.Errorf("send nil slice with shortURLs")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	toDelete := make(map[string]bool)
	for _, shortURL := range shortURLs {
		toDelete[shortURL] = true
	}

	for i, url := range ms.urls {
		if toDelete[url.ShortURL] && url.UserID == userID {
			ms.urls[i].NeedDelete = true
		}
	}

	return nil
}
