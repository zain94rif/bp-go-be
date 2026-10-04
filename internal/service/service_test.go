package service

import (
	"context"
	"testing"

	"bpjs-be/internal/model"
	"bpjs-be/internal/repository"
)

func TestValidateEmployee(t *testing.T) {
	phone, email, place, date, address := "08123456789", "budi@example.com", "Jakarta", "1990-01-01", "Jl. Merdeka"
	valid := model.EmployeeInput{NIK: "1234567890123456", KPJ: "KPJ-1", FullName: "Budi Santoso", Phone: &phone, Email: &email, BirthPlace: &place, BirthDate: &date, Address: &address}
	if err := ValidateEmployee(valid); err != nil {
		t.Fatalf("valid employee rejected: %v", err)
	}

	tests := []model.EmployeeInput{
		{NIK: "123", KPJ: "KPJ", FullName: "Name"},
		{NIK: "1234567890123456", FullName: "Name"},
		{NIK: "1234567890123456", KPJ: "KPJ", FullName: ""},
		{NIK: "1234567890123456", KPJ: "KPJ", FullName: "Name", Email: stringPtr("not-an-email")},
	}
	for _, in := range tests {
		if err := ValidateEmployee(in); err == nil {
			t.Errorf("expected validation error for %+v", in)
		}
	}
}

func TestSearchFieldValidation(t *testing.T) {
	svc := &Service{Repo: repository.Unavailable{}}
	if _, _, err := svc.ListEmployees(context.Background(), "ahmad", "full_name", 1, 20); err == nil {
		t.Fatal("expected repository error")
	}
	if _, _, err := svc.ListEmployees(context.Background(), "ahmad", "unknown", 1, 20); err != ErrValidation {
		t.Fatalf("expected invalid search field error, got %v", err)
	}
}
