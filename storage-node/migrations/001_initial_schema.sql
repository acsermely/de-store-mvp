-- Initial schema for Storage Node
-- Configuration table
CREATE TABLE IF NOT EXISTS config (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stored chunks table
CREATE TABLE IF NOT EXISTS stored_chunks (
    id VARCHAR(64) PRIMARY KEY,
    file_id VARCHAR(64) NOT NULL,
    chunk_index INTEGER NOT NULL,
    hash VARCHAR(64) NOT NULL,
    size_bytes INTEGER NOT NULL,
    file_path VARCHAR(512) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Proof responses history
CREATE TABLE IF NOT EXISTS proof_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chunk_id VARCHAR(64) NOT NULL,
    challenge_id VARCHAR(64) NOT NULL,
    proof_hash VARCHAR(64) NOT NULL,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_stored_chunks_file_id ON stored_chunks(file_id);
CREATE INDEX idx_stored_chunks_status ON stored_chunks(status);
CREATE INDEX idx_proof_history_chunk_id ON proof_history(chunk_id);