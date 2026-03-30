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
	ErrConflict                = errors.New("url already exists")

)

type Repository interface {
	Save(url *model.URL) error
	GetByShortURL(shortURL string) (*model.URL, error)
	GetByOringURL(origURL string) (*model.URL, error)
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