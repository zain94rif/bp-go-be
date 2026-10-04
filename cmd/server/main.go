package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bpjs-be/internal/config"
	"bpjs-be/internal/handler"
	"bpjs-be/internal/middleware"
	"bpjs-be/internal/repository"
	"bpjs-be/internal/service"
	"bpjs-be/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	repo := repository.Repository(repository.Unavailable{})
	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		parsed, err := pgxpool.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			log.Printf("database configuration invalid: %v", err)
		} else if !config.ValidDatabaseSchema(cfg.DatabaseSchema) {
			log.Printf("database schema invalid: %q", cfg.DatabaseSchema)
		} else {
			if parsed.ConnConfig.RuntimeParams == nil {
				parsed.ConnConfig.RuntimeParams = make(map[string]string)
			}
			parsed.ConnConfig.RuntimeParams["search_path"] = cfg.DatabaseSchema
			pool, err = pgxpool.NewWithConfig(context.Background(), parsed)
			if err != nil {
				log.Printf("database pool unavailable: %v", err)
			} else {
				repo = &repository.Postgres{Pool: pool}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if err := repo.Ping(ctx); err != nil {
					log.Printf("database is not ready: %v", err)
				}
				cancel()
			}
		}
	} else {
		log.Printf("DATABASE_URL is not configured; readiness will fail")
	}
	if pool != nil {
		defer pool.Close()
	}

	svc := &service.Service{Repo: repo, JWTSecret: cfg.JWTSecret, AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL,
		CAPTCHARequired: cfg.CAPTCHARequired, CAPTCHAVerifyURL: cfg.CAPTCHAVerifyURL, CAPTCHASEcret: cfg.CAPTCHASEcret, CAPTCHAMode: cfg.CAPTCHAMode,
		MaxUploadBytes: cfg.MaxUploadBytes}
	if svc.AccessTTL <= 0 {
		svc.AccessTTL = 15 * time.Minute
	}
	if svc.RefreshTTL <= 0 {
		svc.RefreshTTL = 720 * time.Hour
	}
	_ = svc.CreateSeedAdmin(context.Background(), cfg.SeedAdminEmail, cfg.SeedAdminPassword)
	h := &handler.Handler{Service: svc, Storage: storage.Local{Root: cfg.StoragePath}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("GET /api/v1/auth/captcha", h.NewCAPTCHA)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	authenticated := func(next http.Handler) http.Handler { return middleware.Auth(cfg.JWTSecret, next) }
	readOnly := func(next http.Handler) http.Handler {
		return authenticated(middleware.RequireRole("ADMIN", "VIEWER")(next))
	}
	adminOnly := func(next http.Handler) http.Handler { return authenticated(middleware.RequireRole("ADMIN")(next)) }
	mux.Handle("POST /api/v1/auth/register", adminOnly(http.HandlerFunc(h.Register)))
	mux.Handle("GET /api/v1/users", adminOnly(http.HandlerFunc(h.ListUsers)))
	mux.Handle("POST /api/v1/users", adminOnly(http.HandlerFunc(h.Register)))
	mux.Handle("PUT /api/v1/users", adminOnly(http.HandlerFunc(h.UpdateUser)))
	mux.Handle("DELETE /api/v1/users", adminOnly(http.HandlerFunc(h.DeleteUser)))
	mux.Handle("GET /api/v1/employees", readOnly(http.HandlerFunc(h.ListEmployees)))
	mux.Handle("GET /api/v1/employees/{id}", readOnly(http.HandlerFunc(h.GetEmployee)))
	mux.Handle("GET /api/v1/employees/{id}/photo", readOnly(http.HandlerFunc(h.ServePhoto)))
	mux.Handle("GET /api/v1/employees/{id}/documents", readOnly(http.HandlerFunc(h.ListDocuments)))
	mux.Handle("GET /api/v1/employees/{id}/documents/{documentID}/preview", readOnly(http.HandlerFunc(h.ServeDocument)))
	mux.Handle("GET /api/v1/employees/{id}/documents/{documentID}/download", readOnly(http.HandlerFunc(h.ServeDocument)))
	mux.Handle("POST /api/v1/auth/logout", authenticated(http.HandlerFunc(h.Logout)))
	mux.Handle("POST /api/v1/employees", adminOnly(http.HandlerFunc(h.CreateEmployee)))
	mux.Handle("PUT /api/v1/employees", adminOnly(http.HandlerFunc(h.UpdateEmployee)))
	mux.Handle("DELETE /api/v1/employees", adminOnly(http.HandlerFunc(h.DeleteEmployee)))
	mux.Handle("POST /api/v1/employees/{id}/documents", adminOnly(http.HandlerFunc(h.CreateDocument)))
	mux.Handle("POST /api/v1/documents", adminOnly(http.HandlerFunc(h.CreateDocument)))
	mux.Handle("PUT /api/v1/documents", adminOnly(http.HandlerFunc(h.UpdateDocument)))
	mux.Handle("DELETE /api/v1/documents", adminOnly(http.HandlerFunc(h.DeleteDocument)))
	mux.Handle("POST /api/v1/employees/{id}/documents/upload", adminOnly(http.HandlerFunc(h.Upload)))
	mux.Handle("POST /api/v1/employees/{id}/photo", adminOnly(http.HandlerFunc(h.UploadPhoto)))

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           loggingMiddleware(middleware.CORS(cfg.FrontendURL, mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server stopped: %v", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(start))
	})
}
