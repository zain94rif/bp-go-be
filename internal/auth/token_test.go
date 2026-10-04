package auth

import (
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	token, err := Sign(Claims{Subject: "user-1", Role: "ADMIN", Expires: time.Now().Add(time.Minute).Unix()}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || claims.Role != "ADMIN" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := Parse(token, "wrong"); err == nil {
		t.Fatal("expected signature failure")
	}
}
