CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nik VARCHAR(16) NOT NULL,
    kpj VARCHAR(50) NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    phone VARCHAR(30),
    email VARCHAR(255),
    birth_place VARCHAR(100),
    birth_date DATE,
    address TEXT,
    photo_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX idx_employees_nik_active ON employees (nik) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_employees_kpj_active ON employees (kpj) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_employees_email_active ON employees (lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX idx_employees_deleted_at ON employees (deleted_at);
