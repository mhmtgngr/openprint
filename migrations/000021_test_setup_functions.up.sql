-- OpenPrint Cloud - Test Setup Helper Functions
-- This migration provides helper functions for test cleanup and data management

-- Function to truncate all tables in the correct order (respecting foreign key dependencies)
CREATE OR REPLACE FUNCTION truncate_all_tables() RETURNS void AS $$
DECLARE
    stmt TEXT;
    tables TEXT[] := ARRAY[
        'job_assignments',
        'job_history',
        'print_jobs',
        'documents',
        'user_sessions',
        'audit_log',
        'printers',
        'agents',
        'users',
        'organizations',
        'api_keys',
        'webhooks',
        'invitations',
        'devices',
        'discovered_printers',
        'agent_events',
        'agent_certificates',
        'enrollment_tokens',
        'usage_stats'
    ];
BEGIN
    -- Disable triggers for faster truncation
    SET session_replication_role = 'replica';

    -- Truncate each table with CASCADE
    FOREACH table IN ARRAY tables
    LOOP
        BEGIN
            EXECUTE format('TRUNCATE TABLE %I CASCADE', table);
        EXCEPTION WHEN undefined_table THEN
            -- Table doesn't exist, skip it
            CONTINUE;
        END;
    END LOOP;

    -- Re-enable triggers
    SET session_replication_role = 'origin';
END;
$$ LANGUAGE plpgsql;

-- Function to get all test data for a specific user email
CREATE OR REPLACE FUNCTION get_user_test_data(p_user_email VARCHAR) RETURNS TABLE(
    job_count BIGINT,
    document_count BIGINT,
    assignment_count BIGINT,
    history_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT COUNT(*) FROM print_jobs WHERE user_email = p_user_email),
        (SELECT COUNT(*) FROM documents WHERE user_email = p_user_email),
        (SELECT COUNT(*) FROM job_assignments WHERE job_id IN (
            SELECT id FROM print_jobs WHERE user_email = p_user_email
        )),
        (SELECT COUNT(*) FROM job_history WHERE job_id IN (
            SELECT id FROM print_jobs WHERE user_email = p_user_email
        ));
END;
$$ LANGUAGE plpgsql;

-- Function to safely drop test data (for cleanup between tests)
CREATE OR REPLACE FUNCTION cleanup_test_data(p_user_email VARCHAR DEFAULT NULL) RETURNS void AS $$
BEGIN
    IF p_user_email IS NOT NULL THEN
        -- Delete job history first
        DELETE FROM job_history
        WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = p_user_email);

        -- Delete job assignments
        DELETE FROM job_assignments
        WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = p_user_email);

        -- Delete print jobs
        DELETE FROM print_jobs WHERE user_email = p_user_email;

        -- Delete documents
        DELETE FROM documents WHERE user_email = p_user_email;
    ELSE
        -- Truncate all tables
        PERFORM truncate_all_tables();
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test organization
CREATE OR REPLACE FUNCTION create_test_organization(p_name VARCHAR DEFAULT 'Test Organization')
RETURNS UUID AS $$
DECLARE
    v_org_id UUID;
    v_slug VARCHAR;
BEGIN
    v_org_id := uuid_generate_v4();
    v_slug := 'test-' || substr(v_org_id::text, 1, 8);

    INSERT INTO organizations (id, name, slug, plan)
    VALUES (v_org_id, p_name, v_slug, 'free')
    RETURNING organizations.id INTO v_org_id;

    RETURN v_org_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test user
CREATE OR REPLACE FUNCTION create_test_user(p_org_id UUID, p_email VARCHAR DEFAULT NULL)
RETURNS UUID AS $$
DECLARE
    v_user_id UUID;
    v_user_email VARCHAR;
BEGIN
    v_user_id := uuid_generate_v4();

    IF p_email IS NULL THEN
        v_user_email := 'test-' || substr(v_user_id::text, 1, 8) || '@example.com';
    ELSE
        v_user_email := p_email;
    END IF;

    INSERT INTO users (id, email, password, first_name, last_name, organization_id, is_active)
    VALUES (
        v_user_id,
        v_user_email,
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- bcrypt hash for "password"
        'Test',
        'User',
        p_org_id,
        true
    )
    RETURNING users.id INTO v_user_id;

    RETURN v_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test agent
CREATE OR REPLACE FUNCTION create_test_agent(p_org_id UUID, p_name VARCHAR DEFAULT 'test-agent')
RETURNS UUID AS $$
DECLARE
    v_agent_id UUID;
BEGIN
    v_agent_id := uuid_generate_v4();

    INSERT INTO agents (id, name, version, os, architecture, hostname, organization_id, status)
    VALUES (
        v_agent_id,
        p_name,
        '1.0.0',
        'linux',
        'x86_64',
        'test-host',
        p_org_id,
        'online'
    )
    RETURNING agents.id INTO v_agent_id;

    RETURN v_agent_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test printer
CREATE OR REPLACE FUNCTION create_test_printer(p_agent_id UUID, p_name VARCHAR DEFAULT 'Test Printer')
RETURNS UUID AS $$
DECLARE
    v_printer_id UUID;
BEGIN
    v_printer_id := uuid_generate_v4();

    INSERT INTO printers (id, name, agent_id, status, capabilities)
    VALUES (
        v_printer_id,
        p_name,
        p_agent_id,
        'online',
        '{"color": true, "duplex": true, "media": ["a4", "letter"]}'::jsonb
    )
    RETURNING printers.id INTO v_printer_id;

    RETURN v_printer_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test document
CREATE OR REPLACE FUNCTION create_test_document(p_user_email VARCHAR DEFAULT NULL)
RETURNS UUID AS $$
DECLARE
    v_document_id UUID;
    v_user_email VARCHAR;
BEGIN
    v_document_id := uuid_generate_v4();

    IF p_user_email IS NULL THEN
        v_user_email := 'test@example.com';
    ELSE
        v_user_email := p_user_email;
    END IF;

    INSERT INTO documents (id, name, content_type, size, checksum, storage_path, user_email)
    VALUES (
        v_document_id,
        'test-document.pdf',
        'application/pdf',
        1024,
        'test-checksum-' || substr(v_document_id::text, 1, 8),
        '/test/path-' || substr(v_document_id::text, 1, 8) || '.pdf',
        v_user_email
    )
    RETURNING documents.id INTO v_document_id;

    RETURN v_document_id;
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions on these functions (useful for test users)
GRANT EXECUTE ON FUNCTION truncate_all_tables() TO PUBLIC;
GRANT EXECUTE ON FUNCTION get_user_test_data(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION cleanup_test_data(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_organization(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_user(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_agent(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_printer(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_document(VARCHAR) TO PUBLIC;
