package service

import (
	"crypto/rand"
	"encoding/base64"
)

func CreateRandomString(len int) (string, error) {
	len = 8
	bytes := make([]byte, len)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:len], nil
}