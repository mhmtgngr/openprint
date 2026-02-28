-- OpenPrint Cloud - Test Setup Helper Functions (Rollback)
-- This removes the test helper functions

DROP FUNCTION IF EXISTS truncate_all_tables() CASCADE;
DROP FUNCTION IF EXISTS get_user_test_data(VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS cleanup_test_data(VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS create_test_organization(VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS create_test_user(UUID, VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS create_test_agent(UUID, VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS create_test_printer(UUID, VARCHAR) CASCADE;
DROP FUNCTION IF EXISTS create_test_document(VARCHAR) CASCADE;
