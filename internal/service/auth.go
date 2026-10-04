package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bpjs-be/internal/apperror"
	"bpjs-be/internal/auth"
	"bpjs-be/internal/model"
	"github.com/dchest/captcha"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Register(ctx context.Context, in model.RegisterInput) (model.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	if email == "" || !strings.Contains(email, "@") || len(email) > 255 ||
		len(in.Password) < 8 || (role != "ADMIN" && role != "VIEWER") {
		return model.User{}, apperror.Validation
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	return s.Repo.CreateUser(ctx, model.User{
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
	})
}

func (s *Service) Login(ctx context.Context, in model.LoginInput, ip string) (model.AuthResponse, error) {
	if !s.verifyLoginCAPTCHA(ctx, in) {
		return model.AuthResponse{}, apperror.CAPTCHAInvalid
	}
	if strings.TrimSpace(in.Email) == "" || in.Password == "" {
		return model.AuthResponse{}, apperror.Validation
	}
	u, err := s.Repo.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return model.AuthResponse{}, apperror.InvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return model.AuthResponse{}, apperror.InvalidCredentials
	}
	now := time.Now()
	access, err := auth.Sign(auth.Claims{Subject: u.ID, Email: u.Email, Role: u.Role, Issued: now.Unix(), Expires: now.Add(s.AccessTTL).Unix()}, s.JWTSecret)
	if err != nil {
		return model.AuthResponse{}, err
	}
	refresh, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return model.AuthResponse{}, err
	}
	if err := s.Repo.SaveRefreshToken(ctx, u.ID, hash, now.Add(s.RefreshTTL)); err != nil {
		return model.AuthResponse{}, err
	}
	_ = s.Repo.CreateAuditLog(ctx, u.ID, "LOGIN", "user", u.ID, ip)
	return model.AuthResponse{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(s.AccessTTL.Seconds()), User: u}, nil
}

func (s *Service) NewCAPTCHA() (string, string, error) {
	id := captcha.NewLen(5)
	var image bytes.Buffer
	if err := captcha.WriteImage(&image, id, 240, 80); err != nil {
		return "", "", err
	}
	return id, "data:image/png;base64," + base64.StdEncoding.EncodeToString(image.Bytes()), nil
}

func (s *Service) verifyLoginCAPTCHA(ctx context.Context, in model.LoginInput) bool {
	switch strings.ToLower(strings.TrimSpace(s.CAPTCHAMode)) {
	case "internal":
		return in.CAPTCHAID != "" && in.CAPTCHAAnswer != "" &&
			captcha.VerifyString(in.CAPTCHAID, in.CAPTCHAAnswer)
	case "disabled":
		return true
	default:
		return s.verifyCAPTCHA(ctx, in.CAPTCHAToken)
	}
}

func (s *Service) Logout(ctx context.Context, refresh string, userID, ip string) error {
	if strings.TrimSpace(refresh) == "" {
		return apperror.Validation
	}
	if err := s.Repo.RevokeRefreshToken(ctx, auth.HashOpaqueToken(refresh)); err != nil {
		return err
	}
	if userID != "" {
		_ = s.Repo.CreateAuditLog(ctx, userID, "LOGOUT", "user", userID, ip)
	}
	return nil
}

func (s *Service) Refresh(ctx context.Context, refresh string) (model.AuthResponse, error) {
	userID, err := s.Repo.ConsumeRefreshToken(ctx, auth.HashOpaqueToken(refresh))
	if err != nil {
		return model.AuthResponse{}, err
	}
	u, err := s.Repo.GetUserByID(ctx, userID)
	if err != nil {
		return model.AuthResponse{}, err
	}
	now := time.Now()
	access, err := auth.Sign(auth.Claims{Subject: u.ID, Email: u.Email, Role: u.Role, Issued: now.Unix(), Expires: now.Add(s.AccessTTL).Unix()}, s.JWTSecret)
	if err != nil {
		return model.AuthResponse{}, err
	}
	newRefresh, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return model.AuthResponse{}, err
	}
	if err := s.Repo.SaveRefreshToken(ctx, u.ID, hash, now.Add(s.RefreshTTL)); err != nil {
		return model.AuthResponse{}, err
	}
	return model.AuthResponse{AccessToken: access, RefreshToken: newRefresh, ExpiresIn: int64(s.AccessTTL.Seconds()), User: u}, nil
}

func (s *Service) CreateSeedAdmin(ctx context.Context, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.Repo.CreateUser(ctx, model.User{Email: strings.ToLower(strings.TrimSpace(email)), Role: "ADMIN", PasswordHash: string(hash)})
	if err == apperror.DuplicateUserEmail {
		return nil
	}
	return err
}

func (s *Service) verifyCAPTCHA(ctx context.Context, token string) bool {
	if !s.CAPTCHARequired {
		return true
	}
	if token == "" || s.CAPTCHAVerifyURL == "" {
		return false
	}
	form := url.Values{}
	form.Set("response", token)
	form.Set("secret", s.CAPTCHASEcret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.CAPTCHAVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	var result struct {
		Success bool `json:"success"`
	}
	return json.NewDecoder(resp.Body).Decode(&result) == nil && result.Success
}
