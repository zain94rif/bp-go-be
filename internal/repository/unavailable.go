package repository

import (
	"context"
	"time"

	"bpjs-be/internal/apperror"
	"bpjs-be/internal/model"
)

type Unavailable struct{}

func (Unavailable) Ping(context.Context) error { return apperror.DatabaseUnavailable }
func (Unavailable) ListEmployees(context.Context, string, string, int, int) ([]model.Employee, int, error) {
	return nil, 0, apperror.DatabaseUnavailable
}
func (Unavailable) GetEmployee(context.Context, string) (model.Employee, error) {
	return model.Employee{}, apperror.DatabaseUnavailable
}
func (Unavailable) CreateEmployee(context.Context, model.EmployeeInput) (model.Employee, error) {
	return model.Employee{}, apperror.DatabaseUnavailable
}
func (Unavailable) UpdateEmployee(context.Context, string, model.EmployeeInput) (model.Employee, error) {
	return model.Employee{}, apperror.DatabaseUnavailable
}
func (Unavailable) SoftDeleteEmployee(context.Context, string) error {
	return apperror.DatabaseUnavailable
}
func (Unavailable) UpdateDocument(context.Context, string, model.DocumentInput) (model.Document, error) {
	return model.Document{}, apperror.DatabaseUnavailable
}
func (Unavailable) DeleteDocument(context.Context, string) error { return apperror.DatabaseUnavailable }
func (Unavailable) ListDocuments(context.Context, string) ([]model.Document, error) {
	return nil, apperror.DatabaseUnavailable
}
func (Unavailable) CreateDocument(context.Context, string, model.DocumentInput) (model.Document, error) {
	return model.Document{}, apperror.DatabaseUnavailable
}
func (Unavailable) GetDocument(context.Context, string, string) (model.Document, error) {
	return model.Document{}, apperror.DatabaseUnavailable
}
func (Unavailable) SetEmployeePhoto(context.Context, string, string) error {
	return apperror.DatabaseUnavailable
}
func (Unavailable) GetUserByEmail(context.Context, string) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) GetUserByID(context.Context, string) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) GetUserByIDAny(context.Context, string) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) CreateUser(context.Context, model.User) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) ListUsers(context.Context) ([]model.User, error) {
	return nil, apperror.DatabaseUnavailable
}
func (Unavailable) UpdateUser(context.Context, string, model.UserInput) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) UpdateUserPassword(context.Context, string, string) (model.User, error) {
	return model.User{}, apperror.DatabaseUnavailable
}
func (Unavailable) DeleteUser(context.Context, string) error { return apperror.DatabaseUnavailable }
func (Unavailable) SaveRefreshToken(context.Context, string, string, time.Time) error {
	return apperror.DatabaseUnavailable
}
func (Unavailable) ConsumeRefreshToken(context.Context, string) (string, error) {
	return "", apperror.DatabaseUnavailable
}
func (Unavailable) RevokeRefreshToken(context.Context, string) error {
	return apperror.DatabaseUnavailable
}
func (Unavailable) CreateAuditLog(context.Context, string, string, string, string, string) error {
	return apperror.DatabaseUnavailable
}
