package middleware

import "net/http"

const (
	allowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	allowHeaders = "Content-Type, Authorization"
)

func CORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin != "" && r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", allowMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if allowedOrigin != "" && r.Header.Get("Origin") == allowedOrigin {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
