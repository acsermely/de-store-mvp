package services

import (
	"fmt"
	"os"
	"time"

	"github.com/federated-storage/storage-node/internal/models"
	"github.com/federated-storage/storage-node/internal/storage"
)

// ChunkService handles chunk storage operations
type ChunkService struct {
	db       *storage.DB
	chunkDir string
}

// NewChunkService creates a new chunk service
func NewChunkService(db *storage.DB, chunkDir string) *ChunkService {
	return &ChunkService{
		db:       db,
		chunkDir: chunkDir,
	}
}

// StoreChunk stores a chunk on disk and in database
func (s *ChunkService) StoreChunk(chunkID, fileID string, chunkIndex int, hash string, data []byte) error {
	// Determine file path (two-level directory structure)
	dirPath := fmt.Sprintf("%s/%s/%s", s.chunkDir, chunkID[:2], chunkID[2:4])
	filePath := fmt.Sprintf("%s/%s", dirPath, chunkID)

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	// Write chunk to disk
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write chunk to disk: %w", err)
	}

	// Store in database
	_, err := s.db.Conn.Exec(
		`INSERT INTO stored_chunks (id, file_id, chunk_index, hash, size_bytes, file_path) 
		 VALUES (?, ?, ?, ?, ?, ?) 
		 ON CONFLICT(id) DO UPDATE SET 
		   file_id = excluded.file_id,
		   chunk_index = excluded.chunk_index,
		   hash = excluded.hash,
		   size_bytes = excluded.size_bytes,
		   file_path = excluded.file_path,
		   updated_at = ?`,
		chunkID, fileID, chunkIndex, hash, len(data), filePath, time.Now())
	if err != nil {
		// Clean up the file if database insert fails
		os.Remove(filePath)
		return fmt.Errorf("failed to store chunk metadata: %w", err)
	}

	return nil
}

// GetChunk retrieves a chunk by ID (metadata only)
func (s *ChunkService) GetChunk(chunkID string) (*models.StoredChunk, error) {
	var chunk models.StoredChunk
	err := s.db.Conn.QueryRow(
		"SELECT id, file_id, chunk_index, hash, size_bytes, file_path, status, created_at, updated_at FROM stored_chunks WHERE id = ?",
		chunkID).Scan(
		&chunk.ID, &chunk.FileID, &chunk.ChunkIndex, &chunk.Hash,
		&chunk.SizeBytes, &chunk.FilePath, &chunk.Status, &chunk.CreatedAt, &chunk.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("chunk not found: %w", err)
	}
	return &chunk, nil
}

// GetChunkData retrieves chunk data from disk
func (s *ChunkService) GetChunkData(chunkID string) ([]byte, error) {
	chunk, err := s.GetChunk(chunkID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(chunk.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk from disk: %w", err)
	}

	return data, nil
}

// ListChunks lists all stored chunks
func (s *ChunkService) ListChunks() ([]models.StoredChunk, error) {
	rows, err := s.db.Conn.Query(
		"SELECT id, file_id, chunk_index, hash, size_bytes, file_path, status, created_at, updated_at FROM stored_chunks WHERE status = 'active'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []models.StoredChunk
	for rows.Next() {
		var chunk models.StoredChunk
		err := rows.Scan(
			&chunk.ID, &chunk.FileID, &chunk.ChunkIndex, &chunk.Hash,
			&chunk.SizeBytes, &chunk.FilePath, &chunk.Status, &chunk.CreatedAt, &chunk.UpdatedAt)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// DeleteChunk marks a chunk as deleted
func (s *ChunkService) DeleteChunk(chunkID string) error {
	_, err := s.db.Conn.Exec(
		"UPDATE stored_chunks SET status = 'deleted', updated_at = ? WHERE id = ?",
		time.Now(), chunkID)
	return err
}

// GetTotalStorage returns total storage used in bytes
func (s *ChunkService) GetTotalStorage() (int64, error) {
	var total int64
	err := s.db.Conn.QueryRow(
		"SELECT COALESCE(SUM(size_bytes), 0) FROM stored_chunks WHERE status = 'active'").Scan(&total)
	return total, err
}

// GetChunkCount returns the number of stored chunks
func (s *ChunkService) GetChunkCount() (int, error) {
	var count int
	err := s.db.Conn.QueryRow(
		"SELECT COUNT(*) FROM stored_chunks WHERE status = 'active'").Scan(&count)
	return count, err
}
