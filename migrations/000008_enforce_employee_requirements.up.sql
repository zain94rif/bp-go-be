ALTER TABLE employees
    ALTER COLUMN phone SET NOT NULL,
    ALTER COLUMN email SET NOT NULL,
    ALTER COLUMN birth_place SET NOT NULL,
    ALTER COLUMN birth_date SET NOT NULL,
    ALTER COLUMN address SET NOT NULL;
CREATE UNIQUE INDEX idx_employees_phone_active ON employees(phone) WHERE deleted_at IS NULL;
DROP INDEX IF EXISTS idx_employees_email_active;
CREATE UNIQUE INDEX idx_employees_email_active ON employees(lower(email)) WHERE deleted_at IS NULL;
ALTER TABLE employee_documents ADD CONSTRAINT employee_documents_type_check CHECK (type = 'diploma');
