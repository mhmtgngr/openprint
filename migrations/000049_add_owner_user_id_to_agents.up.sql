-- OpenPrint Cloud - Add owner_user_id to agents table

-- This migration adds the owner_user_id column to the agents table
-- which was missing from the original schema but is used by the repository code

ALTER TABLE agents ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- Create index for owner lookups
CREATE INDEX IF NOT EXISTS idx_agents_owner_user_id ON agents(owner_user_id) WHERE owner_user_id IS NOT NULL;
