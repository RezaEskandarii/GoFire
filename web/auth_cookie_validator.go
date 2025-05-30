package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

func generateAuthToken(username, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(username))
	signature := mac.Sum(nil)
	token := base64.StdEncoding.EncodeToString([]byte(username)) + "|" + base64.StdEncoding.EncodeToString(signature)
	return token
}

func isValidAuthToken(token, secretKey string) bool {
	parts := strings.Split(token, "|")
	if len(parts) != 2 {
		return false
	}
	usernameBytes, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expectedMac, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(usernameBytes)
	calculatedMac := mac.Sum(nil)

	return hmac.Equal(expectedMac, calculatedMac)
}

func isAuthenticated(r *http.Request, secretKey string) bool {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	if !isValidAuthToken(cookie.Value, secretKey) {
		return false
	}
	return true
}
