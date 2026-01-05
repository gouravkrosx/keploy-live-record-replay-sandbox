-- MySQL initialization script
-- This script runs when the container is first created

-- Ensure we're using the correct character set
ALTER DATABASE marketplace_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create indexes for better query performance (GORM will create tables, we add additional indexes)
-- These will be created after GORM auto-migration

-- Grant all privileges to the application user
GRANT ALL PRIVILEGES ON marketplace_db.* TO 'marketplace'@'%';
FLUSH PRIVILEGES;
