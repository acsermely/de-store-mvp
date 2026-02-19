package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/google/uuid"
)

// ChunkService handles chunk operations
type ChunkService struct {
	db          *storage.DB
	nodeService *NodeService
}

// NewChunkService creates a new chunk service
func NewChunkService(db *storage.DB, nodeService *NodeService) *ChunkService {
	return &ChunkService{db: db, nodeService: nodeService}
}

// StoreChunk stores a chunk and its assignments
func (s *ChunkService) StoreChunk(ctx context.Context, fileID uuid.UUID, chunkIndex int, data []byte, nodeIDs []uuid.UUID) (*models.Chunk, error) {
	// Calculate hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	chunk := &models.Chunk{
		ID:         uuid.New(),
		FileID:     fileID,
		ChunkIndex: chunkIndex,
		Hash:       hashStr,
		SizeBytes:  len(data),
	}

	// Insert chunk with data
	_, err := s.db.Pool.Exec(ctx,
		"INSERT INTO chunks (id, file_id, chunk_index, hash, size_bytes, data) VALUES ($1, $2, $3, $4, $5, $6)",
		chunk.ID, chunk.FileID, chunk.ChunkIndex, chunk.Hash, chunk.SizeBytes, data)
	if err != nil {
		return nil, fmt.Errorf("failed to insert chunk: %w", err)
	}

	// Create assignments
	for _, nodeID := range nodeIDs {
		_, err := s.db.Pool.Exec(ctx,
			"INSERT INTO chunk_assignments (id, chunk_id, node_id) VALUES ($1, $2, $3)",
			uuid.New(), chunk.ID, nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunk assignment: %w", err)
		}
	}

	return chunk, nil
}

// GetChunksByFile retrieves all chunks for a file
func (s *ChunkService) GetChunksByFile(ctx context.Context, fileID uuid.UUID) ([]models.Chunk, error) {
	rows, err := s.db.Pool.Query(ctx,
		"SELECT id, file_id, chunk_index, hash, size_bytes FROM chunks WHERE file_id = $1 ORDER BY chunk_index",
		fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []models.Chunk
	for rows.Next() {
		var chunk models.Chunk
		err := rows.Scan(&chunk.ID, &chunk.FileID, &chunk.ChunkIndex, &chunk.Hash, &chunk.SizeBytes)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// GetChunksByFileWithData retrieves all chunks with data for a file
func (s *ChunkService) GetChunksByFileWithData(ctx context.Context, fileID uuid.UUID) (map[int][]byte, error) {
	rows, err := s.db.Pool.Query(ctx,
		"SELECT chunk_index, data FROM chunks WHERE file_id = $1 ORDER BY chunk_index",
		fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunks := make(map[int][]byte)
	for rows.Next() {
		var chunkIndex int
		var data []byte
		err := rows.Scan(&chunkIndex, &data)
		if err != nil {
			return nil, err
		}
		chunks[chunkIndex] = data
	}
	return chunks, nil
}

// GetChunkAssignments retrieves nodes storing a specific chunk
func (s *ChunkService) GetChunkAssignments(ctx context.Context, chunkID uuid.UUID) ([]models.ChunkAssignment, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT ca.id, ca.chunk_id, ca.node_id, ca.status, ca.created_at, sn.peer_id, sn.address
		 FROM chunk_assignments ca
		 JOIN storage_nodes sn ON ca.node_id = sn.id
		 WHERE ca.chunk_id = $1 AND ca.status = 'active' AND sn.status = 'active'`,
		chunkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []models.ChunkAssignment
	for rows.Next() {
		var ca models.ChunkAssignment
		var peerID, address string
		err := rows.Scan(&ca.ID, &ca.ChunkID, &ca.NodeID, &ca.Status, &ca.CreatedAt, &peerID, &address)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, ca)
	}
	return assignments, nil
}

// SelectNodesForChunks selects nodes for storing chunks (round-robin for MVP)
func (s *ChunkService) SelectNodesForChunks(ctx context.Context, replicaCount int) ([]models.StorageNode, error) {
	nodes, err := s.nodeService.GetAllNodes(ctx)
	if err != nil {
		return nil, err
	}

	if len(nodes) < replicaCount {
		return nil, fmt.Errorf("not enough active nodes (%d available, %d required)", len(nodes), replicaCount)
	}

	// Simple round-robin: return first N nodes
	// In production, implement smarter selection based on capacity, latency, etc.
	return nodes[:replicaCount], nil
}

// EncryptChunk encrypts chunk data using AES-256-GCM
func EncryptChunk(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptChunk decrypts chunk data using AES-256-GCM
func DecryptChunk(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
