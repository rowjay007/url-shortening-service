-- Initialize the database for URL shortening service

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create the short_urls table (GORM will handle this, but this is for reference)
-- This script is mainly for initializing the database with extensions
-- and any initial data if needed

-- You can add initial data here if needed
-- INSERT INTO short_urls (id, url, short_code, access_count, created_at, updated_at)
-- VALUES (uuid_generate_v4(), 'https://example.com', 'example1', 0, now(), now());
