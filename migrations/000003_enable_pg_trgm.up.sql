CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_employees_full_name_trgm ON employees USING GIN (full_name gin_trgm_ops);
CREATE INDEX idx_employees_nik_trgm ON employees USING GIN (nik gin_trgm_ops);
CREATE INDEX idx_employees_kpj_trgm ON employees USING GIN (kpj gin_trgm_ops);
CREATE INDEX idx_employees_email_trgm ON employees USING GIN (email gin_trgm_ops);
