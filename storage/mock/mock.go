package mock

import (
	"fmt"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

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