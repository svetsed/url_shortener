package filestorage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
)

type fileStorage struct {
	filepath string
	urls 	 []model.URL
	mu	 	 sync.RWMutex
	encoder  *json.Encoder
	file 	 *os.File
}

func NewFileStorage(filepath string) (*fileStorage, error) {
	if filepath == "" {
		return nil, fmt.Errorf("empty filepath")
	}
	
	fs := &fileStorage{
		filepath: filepath,
		urls: make([]model.URL, 0),
	}

	f, err := os.OpenFile(fs.filepath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	fs.file = f
	fs.encoder = json.NewEncoder(fs.file)

	if err := fs.load(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *fileStorage) load() error {
	scanner := bufio.NewScanner(fs.file)
	for scanner.Scan() {
		var url model.URL
		err := json.Unmarshal(scanner.Bytes(), &url)
		if err != nil {
			return err
		}

		fs.mu.Lock()
		fs.urls = append(fs.urls, url)
		fs.mu.Unlock()
	}

	return nil
}

func (fs *fileStorage) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}

	return nil
}

func (fs *fileStorage) Save(url *model.URL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.urls == nil {
		return storage.ErrorStorageNotInitialized
	}

	err := fs.encoder.Encode(url)
	if err != nil {
		return err
	}

	fs.urls = append(fs.urls, *url)

	return nil
}

func (fs *fileStorage) GetByShortURL(shortURL string) (*model.URL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	for _, url := range fs.urls {
		if url.ShortURL == shortURL {
			return &url, nil
		}
	}

	return nil, storage.ErrorNotFound
}

func (fs *fileStorage) GetByOringURL(origURL string) (*model.URL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	for _, url := range fs.urls {
		if url.OriginalURL == origURL {
			return &url, nil
		}
	}

	return nil, storage.ErrorNotFound
}