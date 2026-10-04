package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Expires int64  `json:"exp"`
	Issued  int64  `json:"iat"`
}

func Sign(claims Claims, secret string) (string, error) {
	header := encode(map[string]string{"alg": "HS256", "typ": "JWT"})
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := encode(body)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func Parse(token, secret string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return Claims{}, errors.New("invalid token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Subject == "" || claims.Expires < time.Now().Unix() {
		return Claims{}, errors.New("expired token")
	}
	return claims, nil
}

func NewOpaqueToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, HashOpaqueToken(token), nil
}

func HashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func encode(value any) string {
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	default:
		raw, _ = json.Marshal(v)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
