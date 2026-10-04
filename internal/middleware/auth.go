package middleware

import (
	"bpjs-be/internal/apperror"
	"bpjs-be/internal/auth"
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsKey contextKey = "claims"

func Auth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			writeAuthError(w, apperror.Unauthorized)
			return
		}
		claims, err := auth.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), secret)
		if err != nil {
			writeAuthError(w, apperror.TokenInvalid)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeAuthError(w, apperror.Unauthorized)
				return
			}
			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeAuthError(w, apperror.Forbidden)
		})
	}
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(auth.Claims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	code := "UNAUTHORIZED"
	if err == apperror.Forbidden {
		status, code = http.StatusForbidden, "FORBIDDEN"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + strings.ToLower(code) + `"}}`))
}
