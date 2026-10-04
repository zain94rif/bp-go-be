package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bpjs-be/internal/repository"
	"bpjs-be/internal/service"
)

func TestPagination(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=10", nil)
	page, limit, err := pagination(req)
	if err != nil || page != 2 || limit != 10 {
		t.Fatalf("unexpected pagination: page=%d limit=%d err=%v", page, limit, err)
	}
	for _, path := range []string{"/?page=0", "/?limit=101", "/?page=bad"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		if _, _, err := pagination(req); err == nil {
			t.Errorf("expected pagination error for %s", path)
		}
	}
}

func TestHealthHandler(t *testing.T) {
	h := &Handler{Service: &service.Service{Repo: repository.Unavailable{}}}
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
