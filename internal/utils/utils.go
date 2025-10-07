package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func BuildCanonicalString(httpMethod string, httpPath string, body map[string]bool, timestamp string) (string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	bodyString := string(b)

	canonical := fmt.Sprintf(
		"%s.%s.%s.%s",
		timestamp,
		httpMethod,
		httpPath,
		bodyString,
	)

	return canonical, nil
}

func SignCanonical(canonical, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(canonical))

	return hex.EncodeToString(h.Sum(nil))
}
