package handler

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"bpjs-be/internal/apperror"
	"bpjs-be/internal/middleware"
	"bpjs-be/internal/model"
	"bpjs-be/internal/service"
	"bpjs-be/internal/storage"
)

type Handler struct {
	Service *service.Service
	Storage storage.Store
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.Service.Repo.Ping(ctx); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	page, limit, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}
	employees, total, err := h.Service.ListEmployees(r.Context(), r.URL.Query().Get("search"), r.URL.Query().Get("field"), page, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	for i := range employees {
		employees[i].PhotoURL = photoURL(employees[i].ID, employees[i].PhotoPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": employees, "meta": map[string]any{"page": page, "limit": limit, "total": total}})
}

func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var in model.EmployeeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.Service.CreateEmployee(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "CREATE_EMPLOYEE", "employee", e.ID)
	e.PhotoURL = photoURL(e.ID, e.PhotoPath)
	writeJSON(w, http.StatusCreated, map[string]any{"data": e})
}

func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	e, err := h.Service.GetEmployee(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	e.PhotoURL = photoURL(e.ID, e.PhotoPath)
	writeJSON(w, http.StatusOK, map[string]any{"data": e})
}

func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	var in model.EmployeeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.Service.UpdateEmployee(r.Context(), in.ID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "UPDATE_EMPLOYEE", "employee", e.ID)
	e.PhotoURL = photoURL(e.ID, e.PhotoPath)
	writeJSON(w, http.StatusOK, map[string]any{"data": e})
}

func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	id := in.ID
	if err := h.Service.DeleteEmployee(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "DELETE_EMPLOYEE", "employee", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	docs, err := h.Service.ListDocuments(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	for i := range docs {
		docs[i].PreviewURL = "/api/v1/employees/" + id + "/documents/" + docs[i].ID + "/preview"
		docs[i].DownloadURL = "/api/v1/employees/" + id + "/documents/" + docs[i].ID + "/download"
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": docs})
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var in model.DocumentInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.EmployeeID == "" {
		in.EmployeeID = r.PathValue("id")
	}
	d, err := h.Service.CreateDocument(r.Context(), in.EmployeeID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "UPLOAD_DOCUMENT", "employee_document", d.ID)
	d.PreviewURL = "/api/v1/employees/" + d.EmployeeID + "/documents/" + d.ID + "/preview"
	d.DownloadURL = "/api/v1/employees/" + d.EmployeeID + "/documents/" + d.ID + "/download"
	writeJSON(w, http.StatusCreated, map[string]any{"data": d})
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	var in model.DocumentInput
	if !decodeJSON(w, r, &in) {
		return
	}
	d, err := h.Service.UpdateDocument(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "UPDATE_DOCUMENT", "employee_document", d.ID)
	writeJSON(w, http.StatusOK, map[string]any{"data": d})
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.DeleteDocument(r.Context(), in.ID); err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "DELETE_DOCUMENT", "employee_document", in.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in model.LoginInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := h.Service.Login(r.Context(), in, clientIP(r.RemoteAddr))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in model.RegisterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	u, err := h.Service.Register(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "CREATE_USER", "user", u.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"data": u})
}

func (h *Handler) NewCAPTCHA(w http.ResponseWriter, r *http.Request) {
	if h.Service.CAPTCHAMode != "internal" {
		writeError(w, apperror.Validation)
		return
	}
	id, image, err := h.Service.NewCAPTCHA()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"captcha_id": id, "image": image}})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Service.ListUsers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": users})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var in model.UserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	u, err := h.Service.UpdateUser(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "UPDATE_USER", "user", u.ID)
	writeJSON(w, http.StatusOK, map[string]any{"data": u})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.DeleteUser(r.Context(), in.ID); err != nil {
		writeError(w, err)
		return
	}
	h.audit(r, "DELETE_USER", "user", in.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := h.Service.Refresh(r.Context(), in.RefreshToken)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	if err := h.Service.Logout(r.Context(), in.RefreshToken, claims.Subject, clientIP(r.RemoteAddr)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, service.ErrValidation)
		return
	}
	max := h.maxUploadBytes()
	r.Body = http.MaxBytesReader(w, r.Body, max+1024)
	if err := r.ParseMultipartForm(max); err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	defer file.Close()
	key, size, err := h.Storage.Save(r.Context(), file, header.Filename, max)
	if err != nil {
		writeError(w, err)
		return
	}
	mime, err := detectMIME(file)
	if err != nil || !isAllowedUploadMIME(mime) {
		_ = h.Storage.Remove(key)
		writeError(w, service.ErrValidation)
		return
	}
	in := model.DocumentInput{Type: "diploma", FileName: filepath.Base(header.Filename), FilePath: key, MIMEType: mime, FileSize: size}
	doc, err := h.Service.CreateDocument(r.Context(), id, in)
	if err != nil {
		_ = h.Storage.Remove(key)
		writeError(w, err)
		return
	}
	h.audit(r, "UPLOAD_DOCUMENT", "employee_document", doc.ID)
	doc.PreviewURL = "/api/v1/employees/" + id + "/documents/" + doc.ID + "/preview"
	doc.DownloadURL = "/api/v1/employees/" + id + "/documents/" + doc.ID + "/download"
	writeJSON(w, http.StatusCreated, map[string]any{"data": doc})
}

func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, service.ErrValidation)
		return
	}
	max := h.maxUploadBytes()
	r.Body = http.MaxBytesReader(w, r.Body, max+1024)
	if err := r.ParseMultipartForm(max); err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, service.ErrValidation)
		return
	}
	defer file.Close()
	key, size, err := h.Storage.Save(r.Context(), file, header.Filename, max)
	if err != nil || size <= 0 {
		writeError(w, service.ErrValidation)
		return
	}
	mime, err := detectMIME(file)
	if err != nil || (mime != "image/jpeg" && mime != "image/png") {
		_ = h.Storage.Remove(key)
		writeError(w, service.ErrValidation)
		return
	}
	if err := h.Service.Repo.SetEmployeePhoto(r.Context(), id, key); err != nil {
		_ = h.Storage.Remove(key)
		writeError(w, err)
		return
	}
	h.audit(r, "UPLOAD_PHOTO", "employee", id)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"photo_url": "/api/v1/employees/" + id + "/photo", "mime_type": mime, "file_size": size}})
}

func (h *Handler) ServeDocument(w http.ResponseWriter, r *http.Request) {
	employeeID, documentID := r.PathValue("id"), r.PathValue("documentID")
	doc, err := h.Service.Repo.GetDocument(r.Context(), employeeID, documentID)
	if err != nil {
		writeError(w, err)
		return
	}
	f, err := h.Storage.Open(doc.FilePath)
	if err != nil {
		writeError(w, apperror.DocumentNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", doc.MIMEType)
	if filepath.Base(r.URL.Path) == "download" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(doc.FileName)+`"`)
	}
	http.ServeContent(w, r, doc.FileName, doc.CreatedAt, f)
}

func (h *Handler) ServePhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := h.Service.GetEmployee(r.Context(), id)
	if err != nil || e.PhotoPath == nil {
		writeError(w, apperror.DocumentNotFound)
		return
	}
	f, err := h.Storage.Open(*e.PhotoPath)
	if err != nil {
		writeError(w, apperror.DocumentNotFound)
		return
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	_, _ = f.Seek(0, 0)
	w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
	http.ServeContent(w, r, filepath.Base(*e.PhotoPath), e.UpdatedAt, f)
}

func pagination(r *http.Request) (int, int, error) {
	page, limit := 1, 20
	var err error
	if value := r.URL.Query().Get("page"); value != "" {
		page, err = strconv.Atoi(value)
		if err != nil {
			return 0, 0, apperror.Validation
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			return 0, 0, apperror.Validation
		}
	}
	if page < 1 || limit < 1 || limit > 100 {
		return 0, 0, apperror.Validation
	}
	return page, limit, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, apperror.Validation)
		return false
	}
	return true
}

func detectMIME(file multipart.File) (string, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}
	_, _ = file.Seek(0, 0)
	return http.DetectContentType(buf[:n]), nil
}

func isAllowedUploadMIME(mime string) bool {
	return mime == "application/pdf" || mime == "image/jpeg" || mime == "image/png"
}

var idPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func validID(id string) bool { return idPattern.MatchString(id) }

func photoURL(id string, path *string) string {
	if path == nil || *path == "" {
		return ""
	}
	return "/api/v1/employees/" + id + "/photo"
}

func (h *Handler) audit(r *http.Request, action, entity, entityID string) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if ok {
		_ = h.Service.Repo.CreateAuditLog(r.Context(), claims.Subject, action, entity, entityID, clientIP(r.RemoteAddr))
	}
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func (h *Handler) maxUploadBytes() int64 {
	if h.Service.MaxUploadBytes > 0 {
		return h.Service.MaxUploadBytes
	}
	return 25 * 1024 * 1024
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	switch {
	case errors.Is(err, apperror.Validation):
		status, code, message = http.StatusBadRequest, "VALIDATION_ERROR", "invalid request"
	case errors.Is(err, apperror.EmployeeNotFound), errors.Is(err, apperror.DocumentNotFound), errors.Is(err, apperror.UserNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, apperror.DuplicateNIK):
		status, code, message = http.StatusConflict, "DUPLICATE_NIK", "NIK already exists"
	case errors.Is(err, apperror.DuplicateKPJ):
		status, code, message = http.StatusConflict, "DUPLICATE_KPJ", "KPJ already exists"
	case errors.Is(err, apperror.DuplicateEmail):
		status, code, message = http.StatusConflict, "DUPLICATE_EMAIL", "email already exists"
	case errors.Is(err, apperror.DuplicatePhone):
		status, code, message = http.StatusConflict, "DUPLICATE_PHONE", "phone already exists"
	case errors.Is(err, apperror.DuplicateUserEmail):
		status, code, message = http.StatusConflict, "DUPLICATE_USER_EMAIL", "user email already exists"
	case errors.Is(err, apperror.DatabaseUnavailable):
		status, code, message = http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unavailable"
	case errors.Is(err, apperror.InvalidCredentials), errors.Is(err, apperror.CAPTCHAInvalid):
		status, code, message = http.StatusUnauthorized, "UNAUTHORIZED", "authentication failed"
	case errors.Is(err, apperror.Unauthorized), errors.Is(err, apperror.TokenInvalid):
		status, code, message = http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized"
	case errors.Is(err, apperror.Forbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "forbidden"
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
