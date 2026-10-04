package model

import "time"

type Employee struct {
	ID         string     `json:"id"`
	NIK        string     `json:"nik"`
	KPJ        string     `json:"kpj"`
	FullName   string     `json:"full_name"`
	Phone      *string    `json:"phone,omitempty"`
	Email      *string    `json:"email,omitempty"`
	BirthPlace *string    `json:"birth_place,omitempty"`
	BirthDate  *time.Time `json:"birth_date,omitempty"`
	Address    *string    `json:"address,omitempty"`
	PhotoPath  *string    `json:"photo_path,omitempty"`
	PhotoURL   string     `json:"photo_url,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Document struct {
	ID          string    `json:"id"`
	EmployeeID  string    `json:"employee_id"`
	Type        string    `json:"type"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	MIMEType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	CreatedAt   time.Time `json:"created_at"`
	PreviewURL  string    `json:"preview_url,omitempty"`
	DownloadURL string    `json:"download_url,omitempty"`
}

type EmployeeInput struct {
	ID         string  `json:"id,omitempty"`
	NIK        string  `json:"nik"`
	KPJ        string  `json:"kpj"`
	FullName   string  `json:"full_name"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	BirthPlace *string `json:"birth_place"`
	BirthDate  *string `json:"birth_date"`
	Address    *string `json:"address"`
}

type DocumentInput struct {
	ID         string `json:"id,omitempty"`
	EmployeeID string `json:"employee_id,omitempty"`
	Type       string `json:"type"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"file_path"`
	MIMEType   string `json:"mime_type"`
	FileSize   int64  `json:"file_size"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserInput struct {
	ID       string `json:"id,omitempty"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Active   *bool  `json:"active,omitempty"`
	Password string `json:"password,omitempty"`
}

type LoginInput struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CAPTCHAToken  string `json:"captcha_token"`
	CAPTCHAID     string `json:"captcha_id"`
	CAPTCHAAnswer string `json:"captcha_answer"`
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}
