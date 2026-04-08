package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

func CreateSignedUserID(userID string) (string, error) {
	secretKey := os.Getenv("SECRET_COOKIE")
	if secretKey == "" {
		return "", fmt.Errorf("secret key don't set")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))[:32]
	return userID + ":" + signature,  nil
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

func GetOrCreateUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie("user_id")
	if err == nil {
		userID, ok := VerifySignedUserID(cookie.Value)
		if ok {
			return userID, nil // Валидная кука, возвращаем userID
		}
	}

	newUserID := uuid.New().String()

	signedValue, err := CreateSignedUserID(newUserID)
	if err != nil {
		return "", fmt.Errorf("failed to create userID")
	}

	http.SetCookie(w, &http.Cookie{
		Name : "user_id",
		Value: signedValue,
		Path : "/",
	})

	return newUserID, nil
}