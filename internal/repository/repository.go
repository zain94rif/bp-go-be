package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bpjs-be/internal/apperror"
	"bpjs-be/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Ping(context.Context) error
	ListEmployees(context.Context, string, string, int, int) ([]model.Employee, int, error)
	GetEmployee(context.Context, string) (model.Employee, error)
	CreateEmployee(context.Context, model.EmployeeInput) (model.Employee, error)
	UpdateEmployee(context.Context, string, model.EmployeeInput) (model.Employee, error)
	SoftDeleteEmployee(context.Context, string) error
	UpdateDocument(context.Context, string, model.DocumentInput) (model.Document, error)
	DeleteDocument(context.Context, string) error
	ListDocuments(context.Context, string) ([]model.Document, error)
	CreateDocument(context.Context, string, model.DocumentInput) (model.Document, error)
	GetDocument(context.Context, string, string) (model.Document, error)
	SetEmployeePhoto(context.Context, string, string) error
	GetUserByEmail(context.Context, string) (model.User, error)
	GetUserByID(context.Context, string) (model.User, error)
	GetUserByIDAny(context.Context, string) (model.User, error)
	CreateUser(context.Context, model.User) (model.User, error)
	ListUsers(context.Context) ([]model.User, error)
	UpdateUser(context.Context, string, model.UserInput) (model.User, error)
	UpdateUserPassword(context.Context, string, string) (model.User, error)
	DeleteUser(context.Context, string) error
	SaveRefreshToken(context.Context, string, string, time.Time) error
	ConsumeRefreshToken(context.Context, string) (string, error)
	RevokeRefreshToken(context.Context, string) error
	CreateAuditLog(context.Context, string, string, string, string, string) error
}

type Postgres struct{ Pool *pgxpool.Pool }

func (r *Postgres) Ping(ctx context.Context) error {
	if r == nil || r.Pool == nil {
		return apperror.DatabaseUnavailable
	}
	if err := r.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("%w: %v", apperror.DatabaseUnavailable, err)
	}
	return nil
}

func (r *Postgres) ListEmployees(ctx context.Context, search, field string, limit, offset int) ([]model.Employee, int, error) {
	search = strings.TrimSpace(search)
	args := []any{}
	where := "deleted_at IS NULL"
	if search != "" {
		args = append(args, search)
		searchable := map[string]string{
			"nik":         "nik",
			"kpj":         "kpj",
			"full_name":   "full_name",
			"phone":       "phone",
			"email":       "email",
			"birth_place": "birth_place",
			"birth_date":  "birth_date::text",
			"address":     "address",
		}
		if field == "all" {
			columns := make([]string, 0, len(searchable))
			for _, column := range searchable {
				columns = append(columns, column)
			}
			where += ` AND (` + strings.Join(columns, ` ILIKE '%' || $1 || '%' OR `) + ` ILIKE '%' || $1 || '%')`
		} else {
			where += ` AND ` + searchable[field] + ` ILIKE '%' || $1 || '%'`
		}
	}
	var total int
	if err := r.Pool.QueryRow(ctx, "SELECT count(*) FROM employees WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.Pool.Query(ctx, `SELECT id::text, nik, kpj, full_name, phone, email,
		birth_place, birth_date, address, photo_path, created_at, updated_at
		FROM employees WHERE `+where+` ORDER BY full_name ASC, id ASC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make([]model.Employee, 0)
	for rows.Next() {
		var e model.Employee
		if err := rows.Scan(&e.ID, &e.NIK, &e.KPJ, &e.FullName, &e.Phone, &e.Email, &e.BirthPlace,
			&e.BirthDate, &e.Address, &e.PhotoPath, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, e)
	}
	return result, total, rows.Err()
}

func (r *Postgres) GetEmployee(ctx context.Context, id string) (model.Employee, error) {
	var e model.Employee
	err := r.Pool.QueryRow(ctx, `SELECT id::text, nik, kpj, full_name, phone, email,
		birth_place, birth_date, address, photo_path, created_at, updated_at
		FROM employees WHERE id=$1 AND deleted_at IS NULL`, id).Scan(
		&e.ID, &e.NIK, &e.KPJ, &e.FullName, &e.Phone, &e.Email, &e.BirthPlace,
		&e.BirthDate, &e.Address, &e.PhotoPath, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Employee{}, apperror.EmployeeNotFound
	}
	return e, err
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", *value)
	return &d, err
}

func (r *Postgres) CreateEmployee(ctx context.Context, in model.EmployeeInput) (model.Employee, error) {
	birthDate, err := parseDate(in.BirthDate)
	if err != nil {
		return model.Employee{}, err
	}
	var e model.Employee
	err = r.Pool.QueryRow(ctx, `INSERT INTO employees
		(nik, kpj, full_name, phone, email, birth_place, birth_date, address)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, nik, kpj, full_name, phone, email, birth_place, birth_date,
		address, photo_path, created_at, updated_at`,
		in.NIK, in.KPJ, in.FullName, in.Phone, in.Email, in.BirthPlace, birthDate, in.Address).
		Scan(&e.ID, &e.NIK, &e.KPJ, &e.FullName, &e.Phone, &e.Email, &e.BirthPlace, &e.BirthDate,
			&e.Address, &e.PhotoPath, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return model.Employee{}, mapConstraintError(err)
	}
	return e, nil
}

func (r *Postgres) UpdateEmployee(ctx context.Context, id string, in model.EmployeeInput) (model.Employee, error) {
	birthDate, err := parseDate(in.BirthDate)
	if err != nil {
		return model.Employee{}, err
	}
	var e model.Employee
	err = r.Pool.QueryRow(ctx, `UPDATE employees SET nik=$1, kpj=$2, full_name=$3, phone=$4,
		email=$5, birth_place=$6, birth_date=$7, address=$8, updated_at=NOW()
		WHERE id=$9 AND deleted_at IS NULL
		RETURNING id::text, nik, kpj, full_name, phone, email, birth_place, birth_date,
		address, photo_path, created_at, updated_at`,
		in.NIK, in.KPJ, in.FullName, in.Phone, in.Email, in.BirthPlace, birthDate, in.Address, id).
		Scan(&e.ID, &e.NIK, &e.KPJ, &e.FullName, &e.Phone, &e.Email, &e.BirthPlace, &e.BirthDate,
			&e.Address, &e.PhotoPath, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Employee{}, apperror.EmployeeNotFound
	}
	if err != nil {
		return model.Employee{}, mapConstraintError(err)
	}
	return e, nil
}

func (r *Postgres) SoftDeleteEmployee(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE employees SET deleted_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.EmployeeNotFound
	}
	return nil
}

func (r *Postgres) UpdateDocument(ctx context.Context, id string, in model.DocumentInput) (model.Document, error) {
	var d model.Document
	err := r.Pool.QueryRow(ctx, `UPDATE employee_documents SET type=$1, file_name=$2, mime_type=$3, file_size=$4
		WHERE id=$5
		RETURNING id::text, employee_id::text, type, file_name, file_path, mime_type, file_size, created_at`,
		in.Type, in.FileName, in.MIMEType, in.FileSize, id).
		Scan(&d.ID, &d.EmployeeID, &d.Type, &d.FileName, &d.FilePath, &d.MIMEType, &d.FileSize, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Document{}, apperror.DocumentNotFound
	}
	return d, err
}

func (r *Postgres) DeleteDocument(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM employee_documents WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.DocumentNotFound
	}
	return nil
}

func (r *Postgres) ListDocuments(ctx context.Context, employeeID string) ([]model.Document, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id::text, employee_id::text, type, file_name,
		file_path, mime_type, file_size, created_at FROM employee_documents
		WHERE employee_id=$1 ORDER BY created_at DESC`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := make([]model.Document, 0)
	for rows.Next() {
		var d model.Document
		if err := rows.Scan(&d.ID, &d.EmployeeID, &d.Type, &d.FileName, &d.FilePath, &d.MIMEType, &d.FileSize, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (r *Postgres) CreateDocument(ctx context.Context, employeeID string, in model.DocumentInput) (model.Document, error) {
	var d model.Document
	err := r.Pool.QueryRow(ctx, `INSERT INTO employee_documents
		(employee_id, type, file_name, file_path, mime_type, file_size)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id::text, employee_id::text, type, file_name, file_path, mime_type, file_size, created_at`,
		employeeID, in.Type, in.FileName, in.FilePath, in.MIMEType, in.FileSize).
		Scan(&d.ID, &d.EmployeeID, &d.Type, &d.FileName, &d.FilePath, &d.MIMEType, &d.FileSize, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Document{}, apperror.EmployeeNotFound
	}
	return d, err
}

func (r *Postgres) GetDocument(ctx context.Context, employeeID, id string) (model.Document, error) {
	var d model.Document
	err := r.Pool.QueryRow(ctx, `SELECT id::text, employee_id::text, type, file_name, file_path, mime_type, file_size, created_at
			FROM employee_documents d JOIN employees e ON e.id=d.employee_id AND e.deleted_at IS NULL
			WHERE d.employee_id=$1 AND d.id=$2`, employeeID, id).Scan(
		&d.ID, &d.EmployeeID, &d.Type, &d.FileName, &d.FilePath, &d.MIMEType, &d.FileSize, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Document{}, apperror.DocumentNotFound
	}
	return d, err
}

func (r *Postgres) SetEmployeePhoto(ctx context.Context, employeeID, key string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE employees SET photo_path=$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, key, employeeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.EmployeeNotFound
	}
	return nil
}

func (r *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `SELECT id::text,email,role,active,password_hash,created_at FROM users WHERE lower(email)=lower($1) AND active`, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, apperror.UserNotFound
	}
	return u, err
}

func (r *Postgres) GetUserByID(ctx context.Context, id string) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `SELECT id::text,email,role,active,password_hash,created_at FROM users WHERE id=$1 AND active`, id).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, apperror.UserNotFound
	}
	return u, err
}

func (r *Postgres) GetUserByIDAny(ctx context.Context, id string) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `SELECT id::text,email,role,active,password_hash,created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, apperror.UserNotFound
	}
	return u, err
}

func (r *Postgres) CreateUser(ctx context.Context, in model.User) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `INSERT INTO users(email,role,password_hash) VALUES($1,$2,$3)
			RETURNING id::text,email,role,active,password_hash,created_at`, in.Email, in.Role, in.PasswordHash).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return model.User{}, mapConstraintError(err)
	}
	return u, nil
}

func (r *Postgres) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id::text,email,role,active,created_at FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Postgres) UpdateUser(ctx context.Context, id string, in model.UserInput) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `UPDATE users SET email=$1, role=$2, active=COALESCE($3, active)
		WHERE id=$4 RETURNING id::text,email,role,active,created_at`, in.Email, in.Role, in.Active, id).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, apperror.UserNotFound
	}
	if err != nil {
		return model.User{}, mapConstraintError(err)
	}
	return u, nil
}

func (r *Postgres) UpdateUserPassword(ctx context.Context, id, passwordHash string) (model.User, error) {
	var u model.User
	err := r.Pool.QueryRow(ctx, `UPDATE users SET password_hash=$1
		WHERE id=$2 RETURNING id::text,email,role,active,created_at`, passwordHash, id).
		Scan(&u.ID, &u.Email, &u.Role, &u.Active, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, apperror.UserNotFound
	}
	return u, err
}

func (r *Postgres) DeleteUser(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE users SET active=false WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.UserNotFound
	}
	return nil
}

func (r *Postgres) SaveRefreshToken(ctx context.Context, userID, hash string, expires time.Time) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, userID, hash, expires)
	return err
}

func (r *Postgres) ConsumeRefreshToken(ctx context.Context, hash string) (string, error) {
	var userID string
	err := r.Pool.QueryRow(ctx, `UPDATE refresh_tokens SET revoked_at=NOW()
			WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW() RETURNING user_id::text`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.TokenInvalid
	}
	return userID, err
}

func (r *Postgres) RevokeRefreshToken(ctx context.Context, hash string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE token_hash=$1 AND revoked_at IS NULL`, hash)
	return err
}

func (r *Postgres) CreateAuditLog(ctx context.Context, userID, action, entity, entityID, ip string) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO audit_logs(user_id,action,entity,entity_id,ip_address) VALUES($1,$2,$3,$4,$5)`,
		userID, action, entity, entityID, ip)
	return err
}
func mapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "idx_employees_nik_active":
			return apperror.DuplicateNIK
		case "idx_employees_kpj_active":
			return apperror.DuplicateKPJ
		case "idx_employees_email_active":
			return apperror.DuplicateEmail
		case "idx_employees_phone_active":
			return apperror.DuplicatePhone
		case "users_email_key":
			return apperror.DuplicateUserEmail
		}
	}
	return err
}
