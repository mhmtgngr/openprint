-- OpenPrint Cloud - Devices Table (User mobile/web devices for push notifications)

CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type VARCHAR(50) NOT NULL CHECK (device_type IN ('ios', 'android', 'web')),
    device_id VARCHAR(255) NOT NULL,
    push_token TEXT,
    push_provider VARCHAR(50) DEFAULT 'fcm' CHECK (push_provider IN ('fcm', 'apns', 'none')),
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_push ON devices(push_token) WHERE push_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_devices_active ON devices(user_id, is_active) WHERE is_active = true;
