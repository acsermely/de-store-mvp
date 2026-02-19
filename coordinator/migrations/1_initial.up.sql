-- Initial schema for Federated Storage Coordinator
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    credits BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Storage nodes table
CREATE TABLE IF NOT EXISTS storage_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    peer_id VARCHAR(255) UNIQUE NOT NULL,
    public_key BYTEA NOT NULL,
    address VARCHAR(255),
    api_key_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    total_storage_bytes BIGINT DEFAULT 0,
    used_storage_bytes BIGINT DEFAULT 0,
    earned_credits BIGINT DEFAULT 0,
    uptime_percentage DECIMAL(5,2) DEFAULT 100.00,
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Files table
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(255),
    encryption_key BYTEA NOT NULL,
    status VARCHAR(50) DEFAULT 'uploading',
    chunk_count INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Chunks table
CREATE TABLE IF NOT EXISTS chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    hash VARCHAR(64) NOT NULL,
    size_bytes INTEGER NOT NULL,
    UNIQUE(file_id, chunk_index)
);

-- Chunk assignments (which node stores which chunk)
CREATE TABLE IF NOT EXISTS chunk_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES storage_nodes(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chunk_id, node_id)
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_storage_nodes_peer_id ON storage_nodes(peer_id);
CREATE INDEX idx_storage_nodes_status ON storage_nodes(status);
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_chunks_file_id ON chunks(file_id);
CREATE INDEX idx_chunk_assignments_chunk_id ON chunk_assignments(chunk_id);
CREATE INDEX idx_chunk_assignments_node_id ON chunk_assignments(node_id);