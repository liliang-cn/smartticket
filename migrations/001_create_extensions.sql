-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- CREATE EXTENSION IF NOT EXISTS "vector"; -- Skip for development, would need pgvector

-- Enable row level security
ALTER DATABASE smartticket SET row_security = on;