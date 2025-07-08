CREATE TYPE payment_status AS ENUM ('PENDING', 'PROCESSED_DEFAULT', 'PROCESSED_FALLBACK', 'FAILED');
CREATE TYPE processor_type AS ENUM ('DEFAULT', 'FALLBACK');

CREATE TABLE payments (
                          correlation_id UUID NOT NULL UNIQUE,
                          amount DECIMAL(10, 2) NOT NULL,
                          status payment_status NOT NULL DEFAULT 'PENDING',
                          processor processor_type,
                          created_at TIMESTAMPTZ NOT NULL
);