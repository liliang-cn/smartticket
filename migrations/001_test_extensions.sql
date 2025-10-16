-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- Note: vector extension will be added later when needed

-- Enable row level security
ALTER DATABASE smartticket SET row_security = on;