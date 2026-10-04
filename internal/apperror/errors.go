package apperror

import "errors"

var (
	EmployeeNotFound    = errors.New("employee not found")
	DuplicateNIK        = errors.New("duplicate nik")
	DuplicateKPJ        = errors.New("duplicate kpj")
	DuplicateEmail      = errors.New("duplicate email")
	DocumentNotFound    = errors.New("document not found")
	Validation          = errors.New("validation error")
	DatabaseUnavailable = errors.New("database unavailable")
	InvalidCredentials  = errors.New("invalid credentials")
	Unauthorized        = errors.New("unauthorized")
	Forbidden           = errors.New("forbidden")
	TokenInvalid        = errors.New("invalid token")
	CAPTCHAInvalid      = errors.New("captcha invalid")
	UserNotFound        = errors.New("user not found")
	DuplicatePhone      = errors.New("duplicate phone")
	DuplicateUserEmail  = errors.New("duplicate user email")
	StorageFailure      = errors.New("storage failure")
)
