package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestVerifyCAPTCHA(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		want       bool
		wantToken  string
		wantSecret string
	}{
		{name: "valid token", response: `{"success":true}`, want: true, wantToken: "valid-token", wantSecret: "test-secret"},
		{name: "invalid token", response: `{"success":false}`, want: false, wantToken: "invalid-token", wantSecret: "test-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("expected POST, got %s", r.Method)
				}
				if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
					t.Fatalf("unexpected content type %q", got)
				}
				if err := r.ParseForm(); err != nil {
					t.Fatalf("parse form: %v", err)
				}
				if got := r.Form.Get("response"); got != tt.wantToken {
					t.Fatalf("unexpected token %q", got)
				}
				if got := r.Form.Get("secret"); got != tt.wantSecret {
					t.Fatalf("unexpected secret %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			svc := &Service{
				CAPTCHARequired:  true,
				CAPTCHAVerifyURL: server.URL,
				CAPTCHASEcret:    tt.wantSecret,
			}
			if got := svc.verifyCAPTCHA(context.Background(), tt.wantToken); got != tt.want {
				t.Fatalf("verifyCAPTCHA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyCAPTCHASkipsWhenDisabled(t *testing.T) {
	svc := &Service{CAPTCHARequired: false}
	if !svc.verifyCAPTCHA(context.Background(), "") {
		t.Fatal("expected CAPTCHA verification to be skipped when disabled")
	}
}

func TestCAPTCHAFormValues(t *testing.T) {
	values := url.Values{}
	values.Set("response", "token")
	values.Set("secret", "secret")
	if values.Encode() == "" {
		t.Fatal("expected encoded CAPTCHA form")
	}
}
