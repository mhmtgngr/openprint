-- Rollback print policies migration

DROP FUNCTION IF EXISTS evaluate_print_policies CASCADE;

DROP TABLE IF EXISTS print_policy_evaluations;
DROP TABLE IF EXISTS print_policy_assignments;
DROP TABLE IF EXISTS print_policy_rules;
DROP TABLE IF EXISTS print_policies;
