package service

import (
	"strings"
	"testing"
)

func TestNewCAPTCHA(t *testing.T) {
	svc := &Service{CAPTCHAMode: "internal"}
	id, image, err := svc.NewCAPTCHA()
	if err != nil {
		t.Fatalf("NewCAPTCHA() error = %v", err)
	}
	if id == "" || !strings.HasPrefix(image, "data:image/png;base64,") {
		t.Fatalf("unexpected CAPTCHA response: id=%q image prefix=%q", id, image[:min(24, len(image))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
