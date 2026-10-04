# BPJS Employee Backend

Backend REST API untuk mengelola data employee dan metadata file/dokumen
employee. Aplikasi dibuat dengan Go dan PostgreSQL. PostgreSQL tetap menjadi
source of truth, sedangkan pencarian menggunakan query terparameterisasi
dengan `ILIKE`.

## Fitur utama

- Menambah, melihat, mengubah, dan melakukan soft delete employee.
- Mencari employee secara umum atau berdasarkan field tertentu.
- Pagination pada daftar employee.
- Upload multipart foto dan dokumen diploma dengan penyimpanan lokal yang aman,
  preview dan download terproteksi.
- Login JWT, refresh token sekali pakai, logout/revocation, role ADMIN/VIEWER,
  CAPTCHA yang dapat diverifikasi melalui HTTP provider, dan audit log.
- Health check dan readiness check PostgreSQL.
- Validasi input dan perlindungan terhadap SQL injection.
- Semua field employee wajib kecuali foto; NIK, KPJ, phone, dan email (case
  insensitive) unik untuk employee aktif.

## Persyaratan

- Go 1.27 atau lebih baru.
- PostgreSQL 14 atau lebih baru.
- Docker dan Docker Compose (opsional).
- `golang-migrate` (opsional, jika menjalankan migration manual).

## Konfigurasi

Salin file environment:

```bash
cp .env.example .env
```

Contoh konfigurasi:

```env
APP_ENV=development
APP_PORT=8080
FRONTEND_URL=http://localhost:5173
DATABASE_URL=postgres://postgres:postgres@localhost:5432/employee_db?sslmode=disable
DATABASE_SCHEMA=ujicoba
STORAGE_DRIVER=local
STORAGE_PATH=./storage
JWT_SECRET=change-me-to-a-long-random-secret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
CAPTCHA_REQUIRED=false
CAPTCHA_MODE=disabled
CAPTCHA_VERIFY_URL=
SEED_ADMIN_EMAIL=admin@example.com
SEED_ADMIN_PASSWORD=change-me
MAX_UPLOAD_BYTES=26214400
```

`DATABASE_SCHEMA` penting jika tabel sudah berada pada schema tertentu. Untuk
kasus tabel berada di schema `ujicoba`, gunakan:

```env
DATABASE_SCHEMA=ujicoba
```

Nilai default `DATABASE_SCHEMA` adalah `public`. Backend mengatur PostgreSQL
`search_path` pada koneksi sehingga query `employees` akan membaca tabel pada
schema yang dikonfigurasi.

`FRONTEND_URL` adalah alamat frontend yang diizinkan mengakses API dari
browser. Contoh untuk frontend Vite:

```env
FRONTEND_URL=http://localhost:5173
```

Nilai ini dibaca dari environment, bukan ditulis di kode backend. Backend
mengirim header CORS berikut untuk origin yang cocok:

```http
Access-Control-Allow-Origin: <FRONTEND_URL>
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

`CAPTCHA_MODE` dapat bernilai `internal`, `provider`, atau `disabled`. Pada
mode `internal`, backend membuat gambar CAPTCHA sendiri:

```http
GET /api/v1/auth/captcha
```

Response berisi `captcha_id` dan gambar PNG dalam format data URL. Kirim
keduanya saat login:

```json
{
  "email": "admin@example.com",
  "password": "password-admin",
  "captcha_id": "<id>",
  "captcha_answer": "<jawaban-di-gambar>"
}
```

Challenge memiliki masa berlaku singkat dan hanya dapat dipakai sekali.

Pada mode `provider`, `CAPTCHA_REQUIRED` default `true`; set ke `false` hanya
untuk development tanpa provider. Jika aktif, login mengirim `captcha_token` ke
`CAPTCHA_VERIFY_URL` menggunakan POST
`application/x-www-form-urlencoded` dengan field `response` dan `secret`, dan
hanya menerima respons `{ "success": true }`. Registrasi publik tidak tersedia;
admin pertama dibuat dari `SEED_ADMIN_EMAIL` dan `SEED_ADMIN_PASSWORD`.

Backend tidak menampilkan widget CAPTCHA. Widget harus dirender oleh FE
menggunakan site key public dari provider. `CAPTCHA_SECRET` hanya berada di
backend. Pada file `.env` saat ini CAPTCHA development dinonaktifkan:

```env
CAPTCHA_REQUIRED=false
```

Untuk mengaktifkan verifikasi nyata, ubah menjadi `true`, isi
`CAPTCHA_VERIFY_URL` dan `CAPTCHA_SECRET`, lalu restart backend. FE harus
mengirim token widget pada `captcha_token`.

## Menjalankan dengan database yang sudah ada

Pastikan database berisi tabel berikut pada schema yang dikonfigurasi:

```text
employees
employee_documents
```

Kemudian jalankan:

```bash
go mod download
go run ./cmd/server
```

Server berjalan pada:

```text
http://localhost:8080
```

Jika struktur tabel belum ada, jalankan migration:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Migration bawaan membuat tabel pada schema default koneksi. Untuk database
dengan schema khusus, buat schema terlebih dahulu dan pastikan migration
dijalankan dengan `search_path` yang sesuai.

## Menjalankan dengan Docker Compose

Untuk menjalankan PostgreSQL baru beserta API:

```bash
docker compose up --build
```

Compose menjalankan service berikut:

- `postgres`: database PostgreSQL.
- `migrate`: menjalankan migration.
- `api`: REST API pada port `8080`.

## Endpoint

Endpoint employee membutuhkan access token. Endpoint `GET` dapat digunakan oleh
`ADMIN` dan `VIEWER`, sedangkan operasi perubahan data dan upload file hanya
dapat digunakan oleh `ADMIN`.

| Method | Endpoint | Keterangan |
|---|---|---|
| GET | `/health` | Mengecek aplikasi hidup |
| GET | `/ready` | Mengecek koneksi PostgreSQL |
| POST | `/api/v1/auth/login` | Login (CAPTCHA jika diwajibkan) |
| GET | `/api/v1/auth/captcha` | Membuat CAPTCHA internal |
| POST | `/api/v1/auth/register` | Registrasi user baru (ADMIN) |
| POST | `/api/v1/auth/refresh` | Rotasi refresh token |
| POST | `/api/v1/auth/logout` | Cabut refresh token |
| GET | `/api/v1/employees` | Daftar/pencarian (ADMIN/VIEWER) |
| POST | `/api/v1/employees` | Membuat employee (ADMIN) |
| GET | `/api/v1/employees/{id}` | Detail employee |
| PUT | `/api/v1/employees` | Mengubah employee, ID di body (ADMIN) |
| DELETE | `/api/v1/employees` | Soft delete, ID di body (ADMIN) |
| GET | `/api/v1/employees/{id}/photo` | Preview foto |
| POST | `/api/v1/employees/{id}/photo` | Upload foto multipart `file` (ADMIN) |
| GET | `/api/v1/employees/{id}/documents` | Daftar dokumen diploma |
| GET | `/api/v1/users` | Daftar user (ADMIN) |
| POST | `/api/v1/users` | Membuat user (ADMIN) |
| PUT | `/api/v1/users` | Mengubah user/password, ID di body (ADMIN) |
| DELETE | `/api/v1/users` | Menonaktifkan user, ID di body (ADMIN) |
| POST | `/api/v1/documents` | Membuat metadata dokumen, employee ID di body (ADMIN) |
| PUT | `/api/v1/documents` | Mengubah dokumen, ID di body (ADMIN) |
| DELETE | `/api/v1/documents` | Menghapus dokumen, ID di body (ADMIN) |
| POST | `/api/v1/employees/{id}/documents/upload` | Upload diploma multipart `file` (ADMIN) |
| GET | `/api/v1/employees/{id}/documents/{documentID}/preview` | Preview dokumen |
| GET | `/api/v1/employees/{id}/documents/{documentID}/download` | Download dokumen |

Spesifikasi OpenAPI tersedia di `docs/openapi.yaml`.

## Migrasi database

Migration harus dijalankan berurutan karena migration terbaru bergantung pada
tabel employee dan user sebelumnya:

```bash
migrate -path migrations -database "$DATABASE_URL" version
migrate -path migrations -database "$DATABASE_URL" up
```

Urutan migration utama:

```text
000001 pgcrypto
000002 employees
000003 pg_trgm
000004 employee_documents
000005 users
000006 refresh_tokens
000007 audit_logs
000008 employee requirements and unique constraints
```

Migration `000008` membuat `phone`, `email`, `birth_place`, `birth_date`, dan
`address` menjadi wajib serta menambahkan unique index phone/email. Sebelum
menjalankan migration tersebut pada database lama, periksa data NULL dan
duplikat:

```sql
SELECT id FROM employees
WHERE phone IS NULL OR email IS NULL OR birth_place IS NULL
   OR birth_date IS NULL OR address IS NULL;

SELECT lower(email), count(*)
FROM employees
WHERE deleted_at IS NULL
GROUP BY lower(email)
HAVING count(*) > 1;

SELECT phone, count(*)
FROM employees
WHERE deleted_at IS NULL
GROUP BY phone
HAVING count(*) > 1;
```

Data bermasalah harus dibersihkan atau dilengkapi terlebih dahulu. Migration
tidak menghapus data otomatis.

## Pendaftaran user pertama

Backend tidak menyediakan public registration. User pertama dibuat saat server
startup menggunakan environment seed:

```env
SEED_ADMIN_EMAIL=admin@example.com
SEED_ADMIN_PASSWORD=ganti-dengan-password-kuat
```

Setelah migration selesai, jalankan server:

```bash
go run ./cmd/server
```

Jika email belum ada, backend membuat user dengan role `ADMIN`. Jika email
sudah ada, seed tidak membuat duplikat. Setelah user pertama berhasil dibuat,
hapus atau kosongkan `SEED_ADMIN_PASSWORD` dari environment dan restart server.
Password disimpan sebagai hash bcrypt dan tidak dikembalikan oleh API.

Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"ganti-dengan-password-kuat","captcha_token":""}'
```

Untuk development tanpa provider CAPTCHA, gunakan `CAPTCHA_REQUIRED=false`.
Untuk production, gunakan `CAPTCHA_REQUIRED=true`, isi `CAPTCHA_VERIFY_URL`
dan `CAPTCHA_SECRET`, lalu kirim token CAPTCHA asli dari frontend.

Response login memberikan `access_token` dan `refresh_token`. Gunakan access
token untuk endpoint protected:

```http
Authorization: Bearer <access_token>
```

Logout mencabut refresh token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'
```

Registrasi user berikutnya dilakukan oleh ADMIN yang sudah login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"email":"viewer@example.com","password":"password-viewer","role":"VIEWER"}'
```

Role yang tersedia adalah `ADMIN` dan `VIEWER`. Password minimal 8 karakter,
email harus unik, dan user baru otomatis aktif.

## Format request employee

Header yang diperlukan:

```http
Content-Type: application/json
```

Contoh request:

```json
{
  "nik": "3578100000000101",
  "kpj": "KPJ-TEST-000101",
  "full_name": "Ahmad Pratama",
  "phone": "081230000101",
  "email": "ahmad.pratama@example.com",
  "birth_place": "Surabaya",
  "birth_date": "1985-01-04",
  "address": "Jl. Contoh No. 1, Surabaya"
}
```

Field wajib:

- `nik`: tepat 16 digit.
- `kpj`: maksimal 50 karakter.
- `full_name`: maksimal 150 karakter.
- `phone`, `email`, `birth_place`, `birth_date`, dan `address`.

Foto tidak dikirim dalam JSON create/update. Upload foto menggunakan endpoint
multipart agar backend menentukan storage key UUID dan MIME sebenarnya.

## Pencarian employee

Pencarian umum mencari pada NIK, KPJ, nama, nomor HP, email, tempat lahir,
tanggal lahir, dan alamat:

```http
GET /api/v1/employees?search=Ahmad
```

Pencarian field tertentu menggunakan parameter `field`:

```http
GET /api/v1/employees?search=3578100000000101&field=nik
GET /api/v1/employees?search=KPJ-TEST&field=kpj
GET /api/v1/employees?search=Ahmad&field=full_name
GET /api/v1/employees?search=0812&field=phone
GET /api/v1/employees?search=example.com&field=email
GET /api/v1/employees?search=Surabaya&field=birth_place
GET /api/v1/employees?search=1985-01-04&field=birth_date
GET /api/v1/employees?search=Jl.%20Contoh&field=address
```

Nilai `field` yang tersedia:

```text
all, nik, kpj, full_name, phone, email, birth_place, birth_date, address
```

Jika `field` tidak diisi, nilainya dianggap `all`. Pagination:

```http
GET /api/v1/employees?search=Surabaya&field=address&page=1&limit=20
```

Default `page` adalah `1`, default `limit` adalah `20`, dan maksimum `limit`
adalah `100`.

Contoh response:

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "nik": "3578100000000101",
      "kpj": "KPJ-TEST-000101",
      "full_name": "Ahmad Pratama",
      "phone": "081230000101",
      "email": "ahmad.pratama@example.com",
      "birth_place": "Surabaya",
      "birth_date": "1985-01-04T00:00:00Z",
      "address": "Jl. Contoh No. 1, Surabaya",
      "photo_path": "employees/photos/photo.jpg",
      "photo_url": "/api/v1/employees/550e8400-e29b-41d4-a716-446655440000/photo",
      "created_at": "2026-09-04T10:00:00Z",
      "updated_at": "2026-09-04T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 1
  }
}
```

## File dan dokumen

Endpoint utama dokumen menerima multipart dan menyimpan binary pada filesystem
lokal, sedangkan PostgreSQL menyimpan metadata. Endpoint JSON lama tetap
tersedia untuk metadata yang sudah disimpan oleh sistem lain:

```http
POST /api/v1/employees/{id}/documents
```

Request:

```json
{
  "type": "diploma",
  "file_name": "ijazah-ahmad.pdf",
  "file_path": "employees/documents/ijazah-ahmad.pdf",
  "mime_type": "application/pdf",
  "file_size": 245760
}
```

MIME type yang didukung adalah `application/pdf`, `image/jpeg`, dan
`image/png`. Ukuran file maksimum adalah 25 MB. Untuk upload baru gunakan:

```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@ijazah-ahmad.pdf" \
  https://localhost:8080/api/v1/employees/{id}/documents/upload
```

Storage key dibuat acak berbasis UUID, nama file tidak dipakai sebagai path,
ukuran dibatasi `MAX_UPLOAD_BYTES`, MIME dideteksi dari isi, dan file dihapus
lagi jika insert metadata gagal. Hanya tipe `diploma` yang diterima.

## Soft delete

Request berikut tidak menghapus row secara permanen:

```http
DELETE /api/v1/employees/{id}
```

Backend mengisi `deleted_at`. Employee yang sudah dihapus tidak muncul pada
daftar, pencarian, detail, atau daftar dokumen normal.

## Status dan error

| Status | Arti |
|---|---|
| 200 | Request berhasil |
| 201 | Data berhasil dibuat |
| 204 | Employee berhasil di-soft-delete |
| 400 | Request atau input tidak valid |
| 404 | Data tidak ditemukan |
| 409 | NIK, KPJ, atau email sudah digunakan |
| 503 | Database tidak tersedia |

Format error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid request"
  }
}
```

## Pengembangan dan validasi

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build ./...
```

## Struktur proyek

```text
cmd/server/       entrypoint aplikasi
internal/config/  konfigurasi environment
internal/handler/ HTTP handler dan response
internal/service/ validasi dan business logic
internal/repository/ query PostgreSQL
internal/model/  model request dan response
migrations/      migration database
docs/             OpenAPI
```
