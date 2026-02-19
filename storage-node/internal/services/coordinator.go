package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/federated-storage/storage-node/internal/config"
)

// CoordinatorClient handles communication with the coordinator
type CoordinatorClient struct {
	config     *config.CoordinatorConfig
	httpClient *http.Client
}

// NewCoordinatorClient creates a new coordinator client
func NewCoordinatorClient(cfg *config.CoordinatorConfig) *CoordinatorClient {
	return &CoordinatorClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterNodeRequest represents node registration request
type RegisterNodeRequest struct {
	Name           string `json:"name"`
	PeerID         string `json:"peer_id"`
	PublicKey      []byte `json:"public_key"`
	Address        string `json:"address"`
	TotalStorageGB int    `json:"total_storage_gb"`
}

// RegisterNodeResponse represents node registration response
type RegisterNodeResponse struct {
	NodeID string `json:"node_id"`
	APIKey string `json:"api_key"`
}

// RegisterNode registers the node with the coordinator
func (c *CoordinatorClient) RegisterNode(req RegisterNodeRequest) (*RegisterNodeResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		c.config.URL+"/api/v1/nodes/register",
		"application/json",
		bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to register node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	var result RegisterNodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// HeartbeatRequest represents heartbeat request
type HeartbeatRequest struct {
	UsedStorageBytes int64 `json:"used_storage_bytes"`
}

// HeartbeatResponse represents heartbeat response
type HeartbeatResponse struct {
	Status        string `json:"status"`
	EarnedCredits int64  `json:"earned_credits"`
}

// SendHeartbeat sends heartbeat to coordinator
func (c *CoordinatorClient) SendHeartbeat(usedBytes int64) (*HeartbeatResponse, error) {
	req := HeartbeatRequest{UsedStorageBytes: usedBytes}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.config.URL+"/api/v1/nodes/heartbeat", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Peer-ID", c.config.PeerID)
	httpReq.Header.Set("X-API-Key", c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	var result HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ProofEngine handles proof-of-storage generation
type ProofEngine struct {
	chunkService *ChunkService
}

// NewProofEngine creates a new proof engine
func NewProofEngine(chunkService *ChunkService) *ProofEngine {
	return &ProofEngine{chunkService: chunkService}
}

// ProofResult represents a generated proof
type ProofResult struct {
	ProofHash  string
	DurationMs int64
}

// GenerateProof generates a storage proof for a chunk
func (e *ProofEngine) GenerateProof(chunkID string, seed []byte, difficulty int) (*ProofResult, error) {
	start := time.Now()

	// Get chunk metadata
	chunk, err := e.chunkService.GetChunk(chunkID)
	if err != nil {
		return nil, fmt.Errorf("chunk not found: %w", err)
	}

	// Generate proof using sequential hashing
	// In a real implementation, this would use the actual chunk data
	data := append(seed, []byte(chunk.Hash)...)

	// Perform sequential hashing based on difficulty
	for i := 0; i < difficulty; i++ {
		hash := sha256.Sum256(data)
		data = hash[:]
	}

	duration := time.Since(start)
	proofHash := hex.EncodeToString(data)

	return &ProofResult{
		ProofHash:  proofHash,
		DurationMs: duration.Milliseconds(),
	}, nil
}

// RecordProof records a proof response in the database
func (e *ProofEngine) RecordProof(ctx context.Context, challengeID, chunkID, proofHash string, durationMs int64) error {
	_, err := e.chunkService.db.Conn.Exec(
		"INSERT INTO proof_history (chunk_id, challenge_id, proof_hash, duration_ms) VALUES (?, ?, ?, ?)",
		chunkID, challengeID, proofHash, durationMs)
	return err
}
