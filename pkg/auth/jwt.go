package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrEmptySecret  = errors.New("jwt secret cannot be empty")
)

type Claims struct {
	Sub  string `json:"sub"`
	Role string `json:"role,omitempty"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type contextKey struct{}

func Sign(secret []byte, claims Claims) (string, error) {
	if len(secret) == 0 {
		return "", ErrEmptySecret
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return "", ErrInvalidToken
	}
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}
	if claims.Exp == 0 {
		claims.Exp = claims.Iat + int64((24 * time.Hour).Seconds())
	}

	headerBytes, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString(headerBytes) + "." + enc.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	sig := enc.EncodeToString(mac.Sum(nil))

	return unsigned + "." + sig, nil
}

func Verify(secret []byte, token string) (Claims, error) {
	if len(secret) == 0 {
		return Claims{}, ErrEmptySecret
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	enc := base64.RawURLEncoding
	headerBytes, err := enc.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var hdr jwtHeader
	if err := json.Unmarshal(headerBytes, &hdr); err != nil || hdr.Alg != "HS256" {
		return Claims{}, ErrInvalidToken
	}

	gotSig, err := enc.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(gotSig, mac.Sum(nil)) {
		return Claims{}, ErrInvalidToken
	}

	payloadBytes, err := enc.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return Claims{}, ErrInvalidToken
	}
	if claims.Exp <= time.Now().Unix() {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(contextKey{}).(Claims)
	return claims, ok
}

func UserID(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || strings.TrimSpace(claims.Sub) == "" {
		return "", false
	}
	return claims.Sub, true
}

func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeUnauthorized(w, "invalid authorization header")
				return
			}

			claims, err := Verify(secret, strings.TrimSpace(parts[1]))
			if err != nil {
				writeUnauthorized(w, err.Error())
				return
			}

			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
