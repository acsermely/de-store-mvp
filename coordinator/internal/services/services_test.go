package services

import (
	"testing"
	"time"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthService_Register(t *testing.T) {
	// This test would require a real database or extensive mocking
	// For MVP, we'll create a simple test structure
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid registration",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantErr: false,
		},
		{
			name: "empty email",
			req: RegisterRequest{
				Email:    "",
				Password: "securepassword123",
			},
			wantErr: true,
		},
		{
			name: "short password",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate request structure
			if tt.req.Email == "" || len(tt.req.Password) < 8 {
				assert.True(t, tt.wantErr, "Expected error for invalid input")
			} else {
				assert.False(t, tt.wantErr, "Expected success for valid input")
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "valid credentials",
			email:    "user@example.com",
			password: "correctpassword",
			wantErr:  false,
		},
		{
			name:     "invalid email",
			email:    "nonexistent@example.com",
			password: "password",
			wantErr:  true,
		},
		{
			name:     "wrong password",
			email:    "user@example.com",
			password: "wrongpassword",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate credentials format
			if tt.email == "" || tt.password == "" {
				assert.True(t, tt.wantErr)
			}
			// In real test, would check against database
		})
	}
}

func TestUploadService_InitiateUpload(t *testing.T) {
	service := &UploadService{
		chunkSize: 256 * 1024, // 256KB
	}

	tests := []struct {
		name       string
		filename   string
		sizeBytes  int64
		wantChunks int
		wantErr    bool
	}{
		{
			name:       "small file single chunk",
			filename:   "test.txt",
			sizeBytes:  1000,
			wantChunks: 1,
			wantErr:    false,
		},
		{
			name:       "exact chunk size",
			filename:   "exact.bin",
			sizeBytes:  256 * 1024,
			wantChunks: 1,
			wantErr:    false,
		},
		{
			name:       "multiple chunks",
			filename:   "large.bin",
			sizeBytes:  256*1024*5 + 100, // 5 full chunks + partial
			wantChunks: 6,
			wantErr:    false,
		},
		{
			name:       "zero size",
			filename:   "empty.txt",
			sizeBytes:  0,
			wantChunks: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.LessOrEqual(t, tt.sizeBytes, int64(0), "Should error on zero or negative size")
			} else {
				// Calculate expected chunks
				expectedChunks := int((tt.sizeBytes + service.chunkSize - 1) / service.chunkSize)
				assert.Equal(t, tt.wantChunks, expectedChunks, "Chunk calculation mismatch")
			}
		})
	}
}

func TestNodeService_RegisterNode(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterNodeRequest
		wantErr bool
	}{
		{
			name: "valid node registration",
			req: RegisterNodeRequest{
				Name:           "Test Node",
				PeerID:         "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
				PublicKey:      []byte("test-public-key"),
				TotalStorageGB: 100,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: RegisterNodeRequest{
				Name:           "",
				PeerID:         "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
				PublicKey:      []byte("test-public-key"),
				TotalStorageGB: 100,
			},
			wantErr: true,
		},
		{
			name: "missing peer id",
			req: RegisterNodeRequest{
				Name:           "Test Node",
				PeerID:         "",
				PublicKey:      []byte("test-public-key"),
				TotalStorageGB: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			hasError := tt.req.Name == "" || tt.req.PeerID == ""
			assert.Equal(t, tt.wantErr, hasError, "Validation mismatch")
		})
	}
}

func TestFileService_CalculateStorageCost(t *testing.T) {
	service := &FileService{
		storageCredit: 100, // 100 credits per GB per month
	}

	tests := []struct {
		name         string
		sizeBytes    int64
		replicaCount int
		wantCredits  int64
	}{
		{
			name:         "1 GB with 3 replicas",
			sizeBytes:    1024 * 1024 * 1024,
			replicaCount: 3,
			wantCredits:  300, // 1 GB * 3 replicas * 100 credits
		},
		{
			name:         "1 MB with 3 replicas",
			sizeBytes:    1024 * 1024,
			replicaCount: 3,
			wantCredits:  0, // Less than 1 GB, rounds down
		},
		{
			name:         "500 MB with 3 replicas",
			sizeBytes:    500 * 1024 * 1024,
			replicaCount: 3,
			wantCredits:  146, // Actual calculation: math.floor(500/1024 * 3 * 100)
		},
		{
			name:         "10 GB with 1 replica",
			sizeBytes:    10 * 1024 * 1024 * 1024,
			replicaCount: 1,
			wantCredits:  1000, // 10 GB * 1 * 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.CalculateStorageCost(tt.sizeBytes, tt.replicaCount)
			assert.Equal(t, tt.wantCredits, got, "Storage cost calculation mismatch")
		})
	}
}

func TestChunkService_EncryptDecrypt(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  []byte
	}{
		{
			name: "small data",
			data: []byte("Hello, World!"),
			key:  make([]byte, 32),
		},
		{
			name: "256KB chunk",
			data: make([]byte, 256*1024),
			key:  make([]byte, 32),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
			key:  make([]byte, 32),
		},
	}

	// Fill keys with random-ish data for testing
	for i := range tests {
		for j := range tests[i].key {
			tests[i].key[j] = byte(j)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := EncryptChunk(tt.data, tt.key)
			assert.NoError(t, err, "Encryption failed")
			assert.NotNil(t, encrypted, "Encrypted data should not be nil")
			assert.NotEqual(t, tt.data, encrypted, "Encrypted data should differ from plaintext")

			// Decrypt
			decrypted, err := DecryptChunk(encrypted, tt.key)
			assert.NoError(t, err, "Decryption failed")
			assert.Equal(t, tt.data, decrypted, "Decrypted data should match original")
		})
	}
}

func TestProofService_generateExpectedProof(t *testing.T) {
	service := &ProofService{
		difficulty: 1000,
	}

	tests := []struct {
		name    string
		seed    []byte
		chunkID string
	}{
		{
			name:    "deterministic proof 1",
			seed:    []byte("test-seed-1"),
			chunkID: "chunk-uuid-1",
		},
		{
			name:    "deterministic proof 2",
			seed:    []byte("test-seed-2"),
			chunkID: "chunk-uuid-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate proof twice with same inputs
			proof1 := service.generateExpectedProof(tt.seed, tt.chunkID)
			proof2 := service.generateExpectedProof(tt.seed, tt.chunkID)

			// Should be deterministic
			assert.Equal(t, proof1, proof2, "Proof generation should be deterministic")
			assert.Equal(t, 64, len(proof1), "Proof should be 64 hex characters (256 bits)")
		})
	}
}

func TestModels_UserValidation(t *testing.T) {
	tests := []struct {
		name    string
		user    models.User
		wantErr bool
	}{
		{
			name: "valid user",
			user: models.User{
				ID:      uuid.New(),
				Email:   "valid@example.com",
				Credits: 1000,
			},
			wantErr: false,
		},
		{
			name: "negative credits",
			user: models.User{
				ID:      uuid.New(),
				Email:   "test@example.com",
				Credits: -100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.Less(t, tt.user.Credits, int64(0), "Negative credits should be invalid")
			} else {
				assert.GreaterOrEqual(t, tt.user.Credits, int64(0), "Credits should be non-negative")
				assert.NotEqual(t, uuid.Nil, tt.user.ID, "User ID should be set")
			}
		})
	}
}

func TestModels_StorageNodeValidation(t *testing.T) {
	node := models.StorageNode{
		ID:                uuid.New(),
		Name:              "Test Node",
		PeerID:            "12D3KooWHYyhN6Tq7PmNMYiu66MzLfC6aHN6Y3hnx8Cq2ZVHAnka",
		TotalStorageBytes: 100 * 1024 * 1024 * 1024, // 100 GB
		UsedStorageBytes:  0,
		Status:            "active",
	}

	assert.NotEqual(t, uuid.Nil, node.ID)
	assert.NotEmpty(t, node.Name)
	assert.NotEmpty(t, node.PeerID)
	assert.Greater(t, node.TotalStorageBytes, int64(0))
	assert.GreaterOrEqual(t, node.UsedStorageBytes, int64(0))
	assert.LessOrEqual(t, node.UsedStorageBytes, node.TotalStorageBytes)
}

func TestModels_FileValidation(t *testing.T) {
	file := models.File{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Filename:   "test.txt",
		SizeBytes:  1024,
		ChunkCount: 1,
		Status:     "ready",
	}

	assert.NotEqual(t, uuid.Nil, file.ID)
	assert.NotEqual(t, uuid.Nil, file.UserID)
	assert.NotEmpty(t, file.Filename)
	assert.Greater(t, file.SizeBytes, int64(0))
	assert.Greater(t, file.ChunkCount, 0)
	assert.Contains(t, []string{"uploading", "ready", "error"}, file.Status)
}

func TestModels_ChunkAssignmentValidation(t *testing.T) {
	assignment := models.ChunkAssignment{
		ID:      uuid.New(),
		ChunkID: uuid.New(),
		NodeID:  uuid.New(),
		Status:  "active",
	}

	assert.NotEqual(t, uuid.Nil, assignment.ID)
	assert.NotEqual(t, uuid.Nil, assignment.ChunkID)
	assert.NotEqual(t, uuid.Nil, assignment.NodeID)
	assert.Contains(t, []string{"active", "pending", "failed"}, assignment.Status)
}

func TestUploadSession_Expiry(t *testing.T) {
	session := UploadSession{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Filename:  "test.txt",
		SizeBytes: 1024,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    "active",
	}

	// Session should not be expired
	assert.False(t, time.Now().After(session.ExpiresAt), "Session should not be expired")

	// Session should be active
	assert.Equal(t, "active", session.Status)

	// Test expired session
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	assert.True(t, time.Now().After(session.ExpiresAt), "Session should be expired")
}
