package storage

import (
	"context"
	"errors"

	"github.com/svetsed/url_shortener/internal/model"
)

var (
	ErrorNotFound 			   = errors.New("not found")
	ErrorNotSupported 		   = errors.New("not supported")
	ErrorStorageNotInitialized = errors.New("storage not initialized")
	ErrURLAlreadyExist         = errors.New("url already exists")
	ErrNoDataForSave		   = errors.New("no data for save")

)

type Repository interface {
	Save(url *model.URL) error
	SaveManyURL(newURLs []*model.URL) error
	GetByShortURL(shortURL string) (*model.URL, error)
	GetByOringURL(origURL string) (*model.URL, error)
	GetUserURLs(userID string) ([]model.URL, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Closer interface {
	Close() error
}

type DBRepository interface {
	Repository
	Pinger
	Closer
}

type FileRepository interface {
	Repository
	Closer
}