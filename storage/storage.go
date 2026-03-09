package storage

import (
	"errors"

	"github.com/svetsed/url_shortener/internal/model"
)

var (
	ErrorNotFound = errors.New("not found")
)

type Repository interface {
	Save(url *model.URL) error
	GetByShortURL(shortURL string) (*model.URL, error)
	GetByOringURL(origURL string) (*model.URL, error)
}