package service

import (
	"context"
	"fmt"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"bpjs-be/internal/model"
	"bpjs-be/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var nikPattern = regexp.MustCompile(`^\d{16}$`)
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Service struct {
	Repo             repository.Repository
	JWTSecret        string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	CAPTCHARequired  bool
	CAPTCHAVerifyURL string
	CAPTCHASEcret    string
	CAPTCHAMode      string
	MaxUploadBytes   int64
}

var searchableEmployeeFields = map[string]struct{}{
	"all": {}, "nik": {}, "kpj": {}, "full_name": {}, "phone": {},
	"email": {}, "birth_place": {}, "birth_date": {}, "address": {},
}

func (s *Service) ListEmployees(ctx context.Context, search, field string, page, limit int) ([]model.Employee, int, error) {
	if len([]rune(search)) > 100 || page < 1 || limit < 1 || limit > 100 {
		return nil, 0, ErrValidation
	}
	field = strings.ToLower(strings.TrimSpace(field))
	if field == "" {
		field = "all"
	}
	if _, ok := searchableEmployeeFields[field]; !ok {
		return nil, 0, ErrValidation
	}
	return s.Repo.ListEmployees(ctx, search, field, limit, (page-1)*limit)
}

func (s *Service) GetEmployee(ctx context.Context, id string) (model.Employee, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return model.Employee{}, ErrValidation
	}
	return s.Repo.GetEmployee(ctx, id)
}

func (s *Service) CreateEmployee(ctx context.Context, in model.EmployeeInput) (model.Employee, error) {
	if err := ValidateEmployee(in); err != nil {
		return model.Employee{}, err
	}
	return s.Repo.CreateEmployee(ctx, in)
}

func (s *Service) UpdateEmployee(ctx context.Context, id string, in model.EmployeeInput) (model.Employee, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return model.Employee{}, ErrValidation
	}
	if err := ValidateEmployee(in); err != nil {
		return model.Employee{}, err
	}
	return s.Repo.UpdateEmployee(ctx, id, in)
}

func (s *Service) DeleteEmployee(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return ErrValidation
	}
	return s.Repo.SoftDeleteEmployee(ctx, id)
}

func (s *Service) UpdateDocument(ctx context.Context, in model.DocumentInput) (model.Document, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(in.ID)) || strings.TrimSpace(in.Type) != "diploma" ||
		strings.TrimSpace(in.FileName) == "" || strings.TrimSpace(in.MIMEType) == "" || in.FileSize <= 0 {
		return model.Document{}, ErrValidation
	}
	return s.Repo.UpdateDocument(ctx, in.ID, in)
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return ErrValidation
	}
	return s.Repo.DeleteDocument(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.Repo.ListUsers(ctx)
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(strings.TrimSpace(id)) {
		return ErrValidation
	}
	return s.Repo.DeleteUser(ctx, id)
}

func (s *Service) UpdateUser(ctx context.Context, in model.UserInput) (model.User, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(in.ID)) {
		return model.User{}, ErrValidation
	}
	if in.Password != "" && len(in.Password) < 8 {
		return model.User{}, ErrValidation
	}
	current, err := s.Repo.GetUserByIDAny(ctx, in.ID)
	if err != nil {
		return model.User{}, err
	}
	if strings.TrimSpace(in.Email) == "" {
		in.Email = current.Email
	}
	if strings.TrimSpace(in.Role) == "" {
		in.Role = current.Role
	}
	if !validUserFields(in.Email, in.Role) {
		return model.User{}, ErrValidation
	}
	if in.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return model.User{}, err
		}
		if _, err := s.Repo.UpdateUser(ctx, in.ID, model.UserInput{ID: in.ID, Email: in.Email, Role: in.Role, Active: in.Active}); err != nil {
			return model.User{}, err
		}
		return s.Repo.UpdateUserPassword(ctx, in.ID, string(hash))
	}
	return s.Repo.UpdateUser(ctx, in.ID, in)
}

func validUserFields(email, role string) bool {
	email = strings.TrimSpace(email)
	role = strings.ToUpper(strings.TrimSpace(role))
	return strings.Contains(email, "@") && len(email) <= 255 && (role == "ADMIN" || role == "VIEWER")
}

func (s *Service) ListDocuments(ctx context.Context, employeeID string) ([]model.Document, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(employeeID)) {
		return nil, ErrValidation
	}
	if _, err := s.Repo.GetEmployee(ctx, employeeID); err != nil {
		return nil, err
	}
	return s.Repo.ListDocuments(ctx, employeeID)
}

func (s *Service) CreateDocument(ctx context.Context, employeeID string, in model.DocumentInput) (model.Document, error) {
	if !uuidPattern.MatchString(strings.TrimSpace(employeeID)) {
		return model.Document{}, ErrValidation
	}
	if strings.ToLower(strings.TrimSpace(in.Type)) != "diploma" || strings.TrimSpace(in.FileName) == "" ||
		strings.TrimSpace(in.FilePath) == "" || strings.TrimSpace(in.MIMEType) == "" || in.FileSize <= 0 {
		return model.Document{}, ErrValidation
	}
	if in.FileSize > 25*1024*1024 || filepath.Base(in.FileName) != in.FileName ||
		strings.ContainsAny(in.FileName, `/\`) || filepath.IsAbs(in.FilePath) ||
		strings.Contains(in.FilePath, "..") || !allowedMIME(in.MIMEType) {
		return model.Document{}, ErrValidation
	}
	if _, err := s.Repo.GetEmployee(ctx, employeeID); err != nil {
		return model.Document{}, err
	}
	return s.Repo.CreateDocument(ctx, employeeID, in)
}

func allowedMIME(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "application/pdf", "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

func ValidateEmployee(in model.EmployeeInput) error {
	if !nikPattern.MatchString(strings.TrimSpace(in.NIK)) {
		return fmt.Errorf("%w: nik must contain exactly 16 digits", ErrValidation)
	}
	if strings.TrimSpace(in.KPJ) == "" || len([]rune(in.KPJ)) > 50 {
		return fmt.Errorf("%w: kpj is required and must be at most 50 characters", ErrValidation)
	}
	if strings.TrimSpace(in.FullName) == "" || len([]rune(in.FullName)) > 150 {
		return fmt.Errorf("%w: full_name is required and must be at most 150 characters", ErrValidation)
	}
	if in.Phone == nil || strings.TrimSpace(*in.Phone) == "" || len([]rune(*in.Phone)) > 30 {
		return fmt.Errorf("%w: phone is required and must be at most 30 characters", ErrValidation)
	}
	if in.Email == nil || strings.TrimSpace(*in.Email) == "" {
		return fmt.Errorf("%w: email is required", ErrValidation)
	}
	if len([]rune(*in.Email)) > 255 {
		return fmt.Errorf("%w: email is too long", ErrValidation)
	}
	if _, err := mail.ParseAddress(*in.Email); err != nil {
		return fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if in.BirthPlace == nil || strings.TrimSpace(*in.BirthPlace) == "" || len([]rune(*in.BirthPlace)) > 100 {
		return fmt.Errorf("%w: birth_place is required and must be at most 100 characters", ErrValidation)
	}
	if in.BirthDate == nil || strings.TrimSpace(*in.BirthDate) == "" {
		return fmt.Errorf("%w: birth_date is required", ErrValidation)
	}
	d, err := time.Parse("2006-01-02", *in.BirthDate)
	if err != nil || d.After(time.Now()) {
		return fmt.Errorf("%w: birth_date must be a valid non-future date", ErrValidation)
	}
	if in.Address == nil || strings.TrimSpace(*in.Address) == "" {
		return fmt.Errorf("%w: address is required", ErrValidation)
	}
	return nil
}

func stringPtr(value string) *string {
	return &value
}
