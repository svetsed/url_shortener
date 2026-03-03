package service

import (
	"crypto/rand"
	"encoding/base64"
)

func CreateRandomString(length int) (string, error) {
	length = 8
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}