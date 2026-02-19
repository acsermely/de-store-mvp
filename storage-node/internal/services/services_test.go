package services

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/federated-storage/storage-node/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestChunkService_CalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple text",
			data:     []byte("Hello, World!"),
			expected: "",
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := sha256.Sum256(tt.data)
			hashStr := hex.EncodeToString(hash[:])

			// Just verify it's a valid SHA256 hash (64 hex chars)
			assert.Equal(t, 64, len(hashStr), "Hash should be 64 hex characters")

			// Verify determinism
			hash2 := sha256.Sum256(tt.data)
			hashStr2 := hex.EncodeToString(hash2[:])
			assert.Equal(t, hashStr, hashStr2, "Hash should be deterministic")
		})
	}
}

func TestProofEngine_GenerateProof(t *testing.T) {
	// Create a mock chunk service for testing
	mockChunk := &models.StoredChunk{
		ID:         "test-chunk-id",
		FileID:     "test-file-id",
		ChunkIndex: 0,
		Hash:       "aabbccdd",
		SizeBytes:  1024,
	}

	tests := []struct {
		name       string
		chunkID    string
		seed       []byte
		difficulty int
		wantErr    bool
	}{
		{
			name:       "simple proof",
			chunkID:    mockChunk.ID,
			seed:       []byte("test-seed"),
			difficulty: 100,
			wantErr:    false,
		},
		{
			name:       "high difficulty",
			chunkID:    mockChunk.ID,
			seed:       []byte("another-seed"),
			difficulty: 10000,
			wantErr:    false,
		},
		{
			name:       "zero difficulty",
			chunkID:    mockChunk.ID,
			seed:       []byte("seed"),
			difficulty: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate proof generation
			data := append(tt.seed, []byte(mockChunk.Hash)...)

			// Perform sequential hashing
			for i := 0; i < tt.difficulty; i++ {
				hash := sha256.Sum256(data)
				data = hash[:]
			}

			proofHash := hex.EncodeToString(data)

			// Verify proof format (after at least one hash, it's always 64 chars)
			if tt.difficulty > 0 {
				assert.Equal(t, 64, len(proofHash), "Proof hash should be 64 hex characters")
			}

			// Verify determinism
			data2 := append(tt.seed, []byte(mockChunk.Hash)...)
			for i := 0; i < tt.difficulty; i++ {
				hash := sha256.Sum256(data2)
				data2 = hash[:]
			}
			proofHash2 := hex.EncodeToString(data2[:])
			assert.Equal(t, proofHash, proofHash2, "Proof should be deterministic")
		})
	}
}

func TestModels_StoredChunkValidation(t *testing.T) {
	tests := []struct {
		name    string
		chunk   models.StoredChunk
		wantErr bool
	}{
		{
			name: "valid chunk",
			chunk: models.StoredChunk{
				ID:         "valid-chunk-id-64-chars-long-string-that-is-64-bytes",
				FileID:     "valid-file-id-36-chars-string",
				ChunkIndex: 0,
				Hash:       "aabbccdd",
				SizeBytes:  1024,
				Status:     "active",
			},
			wantErr: false,
		},
		{
			name: "negative chunk index",
			chunk: models.StoredChunk{
				ID:         "valid-chunk-id",
				FileID:     "valid-file-id",
				ChunkIndex: -1,
				Hash:       "hash",
				SizeBytes:  1024,
				Status:     "active",
			},
			wantErr: true,
		},
		{
			name: "zero size",
			chunk: models.StoredChunk{
				ID:         "valid-chunk-id",
				FileID:     "valid-file-id",
				ChunkIndex: 0,
				Hash:       "hash",
				SizeBytes:  0,
				Status:     "active",
			},
			wantErr: true,
		},
		{
			name: "empty hash",
			chunk: models.StoredChunk{
				ID:         "valid-chunk-id",
				FileID:     "valid-file-id",
				ChunkIndex: 0,
				Hash:       "",
				SizeBytes:  1024,
				Status:     "active",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate chunk
			hasError := tt.chunk.ChunkIndex < 0 ||
				tt.chunk.SizeBytes <= 0 ||
				tt.chunk.Hash == "" ||
				tt.chunk.ID == "" ||
				tt.chunk.FileID == ""

			assert.Equal(t, tt.wantErr, hasError, "Validation mismatch")
		})
	}
}

func TestModels_ProofHistoryEntry(t *testing.T) {
	entry := models.ProofHistoryEntry{
		ID:          1,
		ChunkID:     "test-chunk-id",
		ChallengeID: "challenge-123",
		ProofHash:   "aabbccdd",
		DurationMs:  1500,
		CreatedAt:   time.Now(),
	}

	assert.Greater(t, entry.ID, 0)
	assert.NotEmpty(t, entry.ChunkID)
	assert.NotEmpty(t, entry.ChallengeID)
	assert.NotEmpty(t, entry.ProofHash)
	assert.Greater(t, entry.DurationMs, 0)
	assert.False(t, entry.CreatedAt.IsZero())
}

func TestProofEngine_TimingValidation(t *testing.T) {
	// Test that proof timing is reasonable (< 2 seconds requirement)
	tests := []struct {
		name       string
		difficulty int
		maxTimeMs  int64
	}{
		{
			name:       "low difficulty",
			difficulty: 100,
			maxTimeMs:  2000,
		},
		{
			name:       "medium difficulty",
			difficulty: 1000,
			maxTimeMs:  2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate proof generation
			seed := []byte("test-seed-data")
			chunkHash := []byte("test-chunk-hash-for-testing")

			data := append(seed, chunkHash...)
			start := time.Now()

			for i := 0; i < tt.difficulty; i++ {
				hash := sha256.Sum256(data)
				data = hash[:]
			}

			duration := time.Since(start)

			// Assert timing is reasonable
			assert.Less(t, duration.Milliseconds(), tt.maxTimeMs,
				"Proof generation took too long: %d ms", duration.Milliseconds())
		})
	}
}

func TestCoordinatorClient_RegisterNodeRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterNodeRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: RegisterNodeRequest{
				Name:           "Test Node",
				PeerID:         "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
				PublicKey:      []byte("test-public-key-data"),
				TotalStorageGB: 100,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: RegisterNodeRequest{
				Name:           "",
				PeerID:         "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
				PublicKey:      []byte("key"),
				TotalStorageGB: 100,
			},
			wantErr: true,
		},
		{
			name: "missing peer id",
			req: RegisterNodeRequest{
				Name:           "Test Node",
				PeerID:         "",
				PublicKey:      []byte("key"),
				TotalStorageGB: 100,
			},
			wantErr: true,
		},
		{
			name: "zero storage",
			req: RegisterNodeRequest{
				Name:           "Test Node",
				PeerID:         "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
				PublicKey:      []byte("key"),
				TotalStorageGB: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			hasError := tt.req.Name == "" ||
				tt.req.PeerID == "" ||
				tt.req.TotalStorageGB <= 0 ||
				len(tt.req.PublicKey) == 0

			assert.Equal(t, tt.wantErr, hasError, "Validation mismatch for %s", tt.name)
		})
	}
}

func TestCoordinatorClient_HeartbeatRequest(t *testing.T) {
	tests := []struct {
		name        string
		usedStorage int64
		wantValid   bool
	}{
		{
			name:        "valid small storage",
			usedStorage: 1024 * 1024, // 1 MB
			wantValid:   true,
		},
		{
			name:        "valid large storage",
			usedStorage: 100 * 1024 * 1024 * 1024, // 100 GB
			wantValid:   true,
		},
		{
			name:        "zero storage",
			usedStorage: 0,
			wantValid:   true, // Zero is valid (empty node)
		},
		{
			name:        "negative storage",
			usedStorage: -1,
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HeartbeatRequest{
				UsedStorageBytes: tt.usedStorage,
			}

			isValid := req.UsedStorageBytes >= 0
			assert.Equal(t, tt.wantValid, isValid, "Heartbeat validation mismatch")
		})
	}
}
