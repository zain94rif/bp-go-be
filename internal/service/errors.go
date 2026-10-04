package service

import "bpjs-be/internal/apperror"

var (
	ErrEmployeeNotFound    = apperror.EmployeeNotFound
	ErrDuplicateNIK        = apperror.DuplicateNIK
	ErrDuplicateKPJ        = apperror.DuplicateKPJ
	ErrDuplicateEmail      = apperror.DuplicateEmail
	ErrDocumentNotFound    = apperror.DocumentNotFound
	ErrValidation          = apperror.Validation
	ErrDatabaseUnavailable = apperror.DatabaseUnavailable
	ErrInvalidCredentials  = apperror.InvalidCredentials
	ErrUnauthorized        = apperror.Unauthorized
	ErrForbidden           = apperror.Forbidden
	ErrTokenInvalid        = apperror.TokenInvalid
	ErrCAPTCHAInvalid      = apperror.CAPTCHAInvalid
)
