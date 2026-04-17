package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNoCookies = errors.New("no cookies")
)

func CreateSignedUserID(userID string) (string, error) {
	secretKey := os.Getenv("SECRET_COOKIE")
	if secretKey == "" {
		secretKey = "default-secret-key"
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))[:32]
	return userID + ":" + signature, nil
}

func VerifySignedUserID(signedValue string) (string, bool) {
	parts := strings.SplitN(signedValue, ":", 2)
	if len(parts) != 2 {
		return "", false
	}

	userID := parts[0]
	signature := parts[1]

	expected, err := CreateSignedUserID(userID)
	if err != nil {
		return "", false
	}
	expectedParts := strings.SplitN(expected, ":", 2)
	if len(expectedParts) != 2 {
		return "", false
	}

	return userID, signature == expectedParts[1]
}

func CreateNewUser(w http.ResponseWriter) (string, error) {
	newUserID := uuid.New().String()

	signedValue, err := CreateSignedUserID(newUserID)
	if err != nil {
		return "", fmt.Errorf("failed to create userID: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    signedValue,
		Path:     "/",
		HttpOnly: true,
	})

	return newUserID, nil
}

func GetUserIDFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return "", fmt.Errorf("cookie user_id not found")
	}

	userID, ok := VerifySignedUserID(cookie.Value)
	if !ok {
		return "", fmt.Errorf("invalid cookie signature")
	}

	return userID, nil
}

func GetOrCreateUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	userID, err := GetUserIDFromCookie(r)
	if err == nil {
		return userID, nil
	}

	newUserID, err := CreateNewUser(w)
	if err != nil {
		return "", err
	}

	return newUserID, nil
}
