-- Credit system and transactions
CREATE TABLE IF NOT EXISTS credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    node_id UUID REFERENCES storage_nodes(id) ON DELETE SET NULL,
    transaction_type VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Proof challenges table
CREATE TABLE IF NOT EXISTS proof_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES storage_nodes(id) ON DELETE CASCADE,
    seed BYTEA NOT NULL,
    difficulty INTEGER NOT NULL DEFAULT 1000,
    status VARCHAR(50) DEFAULT 'pending',
    proof_hash VARCHAR(64),
    duration_ms INTEGER,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Node earnings history
CREATE TABLE IF NOT EXISTS node_earnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES storage_nodes(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    storage_bytes BIGINT NOT NULL,
    storage_credits BIGINT NOT NULL,
    uptime_penalty BIGINT DEFAULT 0,
    missed_proof_penalty BIGINT DEFAULT 0,
    total_earnings BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(node_id, date)
);

-- Upload sessions for tracking active uploads
CREATE TABLE IF NOT EXISTS upload_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_id UUID REFERENCES files(id) ON DELETE SET NULL,
    filename VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    encryption_key BYTEA NOT NULL,
    chunk_count INTEGER NOT NULL,
    received_chunks INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_credit_transactions_user_id ON credit_transactions(user_id);
CREATE INDEX idx_credit_transactions_node_id ON credit_transactions(node_id);
CREATE INDEX idx_proof_challenges_node_id ON proof_challenges(node_id);
CREATE INDEX idx_proof_challenges_status ON proof_challenges(status);
CREATE INDEX idx_node_earnings_node_id ON node_earnings(node_id);
CREATE INDEX idx_upload_sessions_user_id ON upload_sessions(user_id);
CREATE INDEX idx_upload_sessions_status ON upload_sessions(status);