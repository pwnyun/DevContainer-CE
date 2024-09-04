package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type TokenPayload struct {
	UserID string `json:"userId"`
}

func GetUserIDFromToken(r *http.Request) (string, error) {
	token := r.Header.Get("X-Forwarded-Access-Token")
	if token == "" {
		return "", errors.New("missing access token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid access token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	var tokenPayload TokenPayload
	if err := json.Unmarshal(payload, &tokenPayload); err != nil {
		return "", err
	}

	if tokenPayload.UserID == "" {
		return "", errors.New("invalid token payload")
	}

	return tokenPayload.UserID, nil
}
