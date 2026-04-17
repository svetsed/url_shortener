package filestorage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/storage"
)

var _ storage.FileRepository = (*fileStorage)(nil)

type fileStorage struct {
	filepath string
	urls     []model.URL
	mu       sync.RWMutex
	encoder  *json.Encoder
	file     *os.File
	dirty    bool
}

func NewFileStorage(filepath string) (*fileStorage, error) {
	if filepath == "" {
		return nil, fmt.Errorf("empty filepath")
	}

	fs := &fileStorage{
		filepath: filepath,
		urls:     make([]model.URL, 0),
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
	if _, err := fs.file.Seek(0, 0); err != nil {
		return err
	}

	scanner := bufio.NewScanner(fs.file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	urls := make([]model.URL, 0)
	for scanner.Scan() {
		var url model.URL
		err := json.Unmarshal(scanner.Bytes(), &url)
		if err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		urls = append(urls, url)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	fs.mu.Lock()
	fs.urls = urls
	fs.mu.Unlock()

	return nil
}

// ----------------   Implement Closer   ----------------

func (fs *fileStorage) Close() error {
	var retErr error
	if err := fs.Flush(); err != nil {
		retErr = fmt.Errorf("flush error: %v", err)
	}

	if fs.file != nil {
		return fs.file.Close()
	}

	return retErr
}

// ---------------- Implement Repository ----------------

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

func (fs *fileStorage) SaveManyURL(newURLs []*model.URL) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.urls == nil {
		return storage.ErrorStorageNotInitialized
	}

	for _, url := range newURLs {
		if err := fs.encoder.Encode(url); err != nil {
			// logger без прерывания?
			return err
		}

		fs.urls = append(fs.urls, *url)
	}

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

func (fs *fileStorage) GetUserURLs(userID string) ([]model.URL, error) {
	userURLs := make([]model.URL, 0)
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for _, url := range fs.urls {
		if url.UserID == userID {
			userURLs = append(userURLs, url)
		}
	}

	return userURLs, nil
}

func (fs *fileStorage) MarkAsDeleted(shortURLs []string, userID string) error {
	if shortURLs == nil {
		return fmt.Errorf("send nil slice with shortURLs")
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	toDelete := make(map[string]bool)
	for _, shortURL := range shortURLs {
		toDelete[shortURL] = true
	}

	changed := false
	for i := range fs.urls {
		if toDelete[fs.urls[i].ShortURL] && fs.urls[i].UserID == userID && !fs.urls[i].NeedDelete {
			fs.urls[i].NeedDelete = true
			changed = true
		}
	}

	if !changed {
		return nil // или ошибку
	}

	fs.dirty = true

	return nil
}

func (fs *fileStorage) Flush() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.dirty {
		return nil
	}

	f, err := os.OpenFile(fs.filepath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, url := range fs.urls {
		if err := encoder.Encode(url); err != nil {
			return err
		}
	}

	return nil
}
