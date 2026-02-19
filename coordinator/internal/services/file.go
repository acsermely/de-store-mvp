package services

import (
	"context"
	"fmt"
	"time"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/google/uuid"
)

// FileService handles file operations
type FileService struct {
	db            *storage.DB
	chunkSize     int64
	storageCredit int64 // credits per GB per month
}

// NewFileService creates a new file service
func NewFileService(db *storage.DB, chunkSize int64, storageCredit int64) *FileService {
	return &FileService{
		db:            db,
		chunkSize:     chunkSize,
		storageCredit: storageCredit,
	}
}

// CreateFile creates a new file record
func (s *FileService) CreateFile(ctx context.Context, userID uuid.UUID, filename string, sizeBytes int64, mimeType string, encryptionKey []byte, chunkCount int) (*models.File, error) {
	file := &models.File{
		ID:            uuid.New(),
		UserID:        userID,
		Filename:      filename,
		SizeBytes:     sizeBytes,
		MimeType:      mimeType,
		EncryptionKey: encryptionKey,
		Status:        "uploading",
		ChunkCount:    chunkCount,
	}

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO files (id, user_id, filename, size_bytes, mime_type, encryption_key, status, chunk_count) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		file.ID, file.UserID, file.Filename, file.SizeBytes, file.MimeType,
		file.EncryptionKey, file.Status, file.ChunkCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// GetFile retrieves a file by ID
func (s *FileService) GetFile(ctx context.Context, fileID uuid.UUID) (*models.File, error) {
	var file models.File
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, user_id, filename, size_bytes, mime_type, encryption_key, status, chunk_count, created_at, updated_at 
		 FROM files WHERE id = $1`,
		fileID).Scan(
		&file.ID, &file.UserID, &file.Filename, &file.SizeBytes, &file.MimeType,
		&file.EncryptionKey, &file.Status, &file.ChunkCount, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("file not found")
	}
	return &file, nil
}

// GetUserFiles retrieves all files for a user
func (s *FileService) GetUserFiles(ctx context.Context, userID uuid.UUID) ([]models.File, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, user_id, filename, size_bytes, mime_type, status, chunk_count, created_at, updated_at 
		 FROM files WHERE user_id = $1 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var f models.File
		err := rows.Scan(
			&f.ID, &f.UserID, &f.Filename, &f.SizeBytes, &f.MimeType,
			&f.Status, &f.ChunkCount, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// MarkFileComplete marks a file as ready
func (s *FileService) MarkFileComplete(ctx context.Context, fileID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx,
		"UPDATE files SET status = 'ready', updated_at = $1 WHERE id = $2",
		time.Now(), fileID)
	return err
}

// DeleteFile deletes a file and its chunks
func (s *FileService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, "DELETE FROM files WHERE id = $1", fileID)
	return err
}

// CalculateStorageCost calculates the storage cost for a file
func (s *FileService) CalculateStorageCost(sizeBytes int64, replicaCount int) int64 {
	// Calculate monthly cost in credits
	// 1 GB = storageCredit credits per month
	gb := float64(sizeBytes*int64(replicaCount)) / (1024 * 1024 * 1024)
	return int64(gb * float64(s.storageCredit))
}
