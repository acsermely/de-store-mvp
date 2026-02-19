package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Credits      int64     `db:"credits" json:"credits"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// StorageNode represents a storage node in the network
type StorageNode struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	Name              string     `db:"name" json:"name"`
	PeerID            string     `db:"peer_id" json:"peer_id"`
	PublicKey         []byte     `db:"public_key" json:"-"`
	Address           string     `db:"address" json:"address"`
	APIKeyHash        string     `db:"api_key_hash" json:"-"`
	Status            string     `db:"status" json:"status"`
	TotalStorageBytes int64      `db:"total_storage_bytes" json:"total_storage_bytes"`
	UsedStorageBytes  int64      `db:"used_storage_bytes" json:"used_storage_bytes"`
	EarnedCredits     int64      `db:"earned_credits" json:"earned_credits"`
	UptimePercentage  float64    `db:"uptime_percentage" json:"uptime_percentage"`
	LastHeartbeat     *time.Time `db:"last_heartbeat" json:"last_heartbeat"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

// File represents a stored file
type File struct {
	ID            uuid.UUID `db:"id" json:"id"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	Filename      string    `db:"filename" json:"filename"`
	SizeBytes     int64     `db:"size_bytes" json:"size_bytes"`
	MimeType      string    `db:"mime_type" json:"mime_type"`
	EncryptionKey []byte    `db:"encryption_key" json:"-"`
	Status        string    `db:"status" json:"status"`
	ChunkCount    int       `db:"chunk_count" json:"chunk_count"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// Chunk represents a file chunk
type Chunk struct {
	ID         uuid.UUID `db:"id" json:"id"`
	FileID     uuid.UUID `db:"file_id" json:"file_id"`
	ChunkIndex int       `db:"chunk_index" json:"chunk_index"`
	Hash       string    `db:"hash" json:"hash"`
	SizeBytes  int       `db:"size_bytes" json:"size_bytes"`
}

// ChunkAssignment represents a chunk stored on a node
type ChunkAssignment struct {
	ID        uuid.UUID `db:"id" json:"id"`
	ChunkID   uuid.UUID `db:"chunk_id" json:"chunk_id"`
	NodeID    uuid.UUID `db:"node_id" json:"node_id"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// UploadSession represents an active upload
type UploadSession struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	UserID         uuid.UUID  `db:"user_id" json:"user_id"`
	FileID         *uuid.UUID `db:"file_id" json:"file_id,omitempty"`
	Filename       string     `db:"filename" json:"filename"`
	SizeBytes      int64      `db:"size_bytes" json:"size_bytes"`
	EncryptionKey  []byte     `db:"encryption_key" json:"-"`
	ChunkCount     int        `db:"chunk_count" json:"chunk_count"`
	ReceivedChunks int        `db:"received_chunks" json:"received_chunks"`
	Status         string     `db:"status" json:"status"`
	ExpiresAt      time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// ProofChallenge represents a storage proof challenge
type ProofChallenge struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	ChunkID    uuid.UUID  `db:"chunk_id" json:"chunk_id"`
	NodeID     uuid.UUID  `db:"node_id" json:"node_id"`
	Seed       []byte     `db:"seed" json:"-"`
	Difficulty int        `db:"difficulty" json:"difficulty"`
	Status     string     `db:"status" json:"status"`
	ProofHash  *string    `db:"proof_hash" json:"proof_hash,omitempty"`
	DurationMs *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	VerifiedAt *time.Time `db:"verified_at" json:"verified_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// CreditTransaction represents a credit transaction
type CreditTransaction struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	UserID          *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	NodeID          *uuid.UUID `db:"node_id" json:"node_id,omitempty"`
	TransactionType string     `db:"transaction_type" json:"transaction_type"`
	Amount          int64      `db:"amount" json:"amount"`
	Description     string     `db:"description" json:"description"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

// NodeEarnings represents daily earnings for a node
type NodeEarnings struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	NodeID             uuid.UUID `db:"node_id" json:"node_id"`
	Date               time.Time `db:"date" json:"date"`
	StorageBytes       int64     `db:"storage_bytes" json:"storage_bytes"`
	StorageCredits     int64     `db:"storage_credits" json:"storage_credits"`
	UptimePenalty      int64     `db:"uptime_penalty" json:"uptime_penalty"`
	MissedProofPenalty int64     `db:"missed_proof_penalty" json:"missed_proof_penalty"`
	TotalEarnings      int64     `db:"total_earnings" json:"total_earnings"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
}
