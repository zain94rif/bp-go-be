ALTER TABLE employee_documents DROP CONSTRAINT IF EXISTS employee_documents_type_check;
DROP INDEX IF EXISTS idx_employees_phone_active;
ALTER TABLE employees
    ALTER COLUMN phone DROP NOT NULL,
    ALTER COLUMN email DROP NOT NULL,
    ALTER COLUMN birth_place DROP NOT NULL,
    ALTER COLUMN birth_date DROP NOT NULL,
    ALTER COLUMN address DROP NOT NULL;
