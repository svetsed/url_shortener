package mock

import (
	"fmt"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

var _ storage.Repository = (*MockStorage)(nil)

// need add mutex
type MockStorage struct {
	urls map[string]*model.URL
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		urls: make(map[string]*model.URL),
	}
}

func (ms *MockStorage) Save(url *model.URL) error {
	if ms.urls == nil {
		return fmt.Errorf("storage not initialized")
	}
	
	if url == nil {
		return fmt.Errorf("nothing save")
	}

	if _, exist := ms.urls[url.ShortURL]; !exist {
		ms.urls[url.ShortURL] = url
	} else {
		return fmt.Errorf("duplicate url")
	}

	return nil
}

func (ms *MockStorage) SaveManyURL(newURLs []*model.URL) error {
	if ms.urls == nil {
		return fmt.Errorf("storage not initialized")
	}
	
	if newURLs == nil {
		return storage.ErrNoDataForSave
	}

	for _, url := range newURLs {
		if _, exist := ms.urls[url.ShortURL]; !exist {
			ms.urls[url.ShortURL] = url
		} else {
			return fmt.Errorf("duplicate url")
		}
	}

	return nil
}

func (ms *MockStorage) GetByShortURL(shortURL string) (*model.URL, error) {
	if url, exist := ms.urls[shortURL]; exist {
		return url, nil
	}

	return nil, storage.ErrorNotFound
}

func (ms *MockStorage) GetByOringURL(origURL string) (*model.URL, error) {
	for _, url := range ms.urls {
		if url.OriginalURL == origURL {
			return url, nil
		}
	}

	return nil, storage.ErrorNotFound
}

func (ms *MockStorage) GetUserURLs(userID string) ([]model.URL, error) {
	// TODO
	return nil, nil
}

func (ms *MockStorage) 	MarkAsDeleted(shortURLs []string, userID string) error {
	// TODO
	return nil
}

func (ms *MockStorage) Close() error {
	return nil
}