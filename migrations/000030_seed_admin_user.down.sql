-- Rollback Seed Default Admin User

-- Remove the default admin user
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001';

-- Remove the default organization
DELETE FROM organizations WHERE id = '00000000-0000-0000-0000-000000000001';
