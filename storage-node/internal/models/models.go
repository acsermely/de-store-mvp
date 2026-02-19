package models

import (
	"time"
)

// StoredChunk represents a chunk stored on this node
type StoredChunk struct {
	ID         string    `db:"id" json:"id"`
	FileID     string    `db:"file_id" json:"file_id"`
	ChunkIndex int       `db:"chunk_index" json:"chunk_index"`
	Hash       string    `db:"hash" json:"hash"`
	SizeBytes  int       `db:"size_bytes" json:"size_bytes"`
	FilePath   string    `db:"file_path" json:"file_path"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// ProofHistoryEntry represents a proof response history entry
type ProofHistoryEntry struct {
	ID          int       `db:"id" json:"id"`
	ChunkID     string    `db:"chunk_id" json:"chunk_id"`
	ChallengeID string    `db:"challenge_id" json:"challenge_id"`
	ProofHash   string    `db:"proof_hash" json:"proof_hash"`
	DurationMs  int       `db:"duration_ms" json:"duration_ms"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ConfigValue represents a configuration key-value pair
type ConfigValue struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
