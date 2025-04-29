-- GRANT ALL PRIVILEGES ON DATABASE premium_caste TO postgres;
-- CREATE DATABASE premium_caste;
SELECT 'CREATE DATABASE premium_caste'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'premium_caste')\gexec

GRANT ALL PRIVILEGES ON DATABASE premium_caste TO postgres;

