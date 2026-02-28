-- Seed Default Admin User
-- This migration creates a default admin account for initial setup
-- IMPORTANT: Change the password after first login!

-- Create default organization for the admin user
INSERT INTO organizations (
    id,
    name,
    slug,
    plan,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Organization',
    'default',
    'enterprise',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Default admin user
-- Email: admin@openprint.local
-- Password: Admin123!
-- The password hash is bcrypt hash of "Admin123!"
INSERT INTO users (
    id,
    email,
    password,
    first_name,
    last_name,
    role,
    organization_id,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@openprint.local',
    '$2a$12$CfucjOqA.F4RHQqYme2Oj.KiwtO9/kM79KftjEhfHuHq7aA7YQpIS', -- bcrypt hash of "Admin123!"
    'System',
    'Administrator',
    'admin',
    '00000000-0000-0000-0000-000000000001', -- Default organization
    true,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;
