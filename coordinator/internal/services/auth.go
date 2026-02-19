package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"time"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication operations
type AuthService struct {
	db *storage.DB
}

// NewAuthService creates a new auth service
func NewAuthService(db *storage.DB) *AuthService {
	return &AuthService{db: db}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Token  string `json:"token"`
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
	// Check if user exists
	var exists bool
	err := s.db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user already exists")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Credits:      0,
	}

	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, credits) 
		 VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.PasswordHash, user.Credits)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*models.User, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx,
		"SELECT id, email, password_hash, credits FROM users WHERE email = $1",
		req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Credits)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &user, nil
}

// GetUser retrieves a user by ID
func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx,
		"SELECT id, email, credits, created_at, updated_at FROM users WHERE id = $1",
		userID).Scan(&user.ID, &user.Email, &user.Credits, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}

// UpdateCredits updates user credits
func (s *AuthService) UpdateCredits(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update user credits
	_, err = tx.Exec(ctx,
		"UPDATE users SET credits = credits + $1, updated_at = $2 WHERE id = $3",
		amount, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update credits: %w", err)
	}

	// Record transaction
	var transactionType string
	if amount >= 0 {
		transactionType = "credit"
	} else {
		transactionType = "debit"
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO credit_transactions (user_id, transaction_type, amount, description) 
		 VALUES ($1, $2, $3, $4)`,
		userID, transactionType, amount, description)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// InitiateUploadRequest represents an upload initiation request
type InitiateUploadRequest struct {
	Filename  string `json:"filename" binding:"required"`
	SizeBytes int64  `json:"size_bytes" binding:"required,min=1"`
	MimeType  string `json:"mime_type"`
}

// InitiateUploadResponse represents an upload initiation response
type InitiateUploadResponse struct {
	SessionID  string `json:"session_id"`
	ChunkCount int    `json:"chunk_count"`
	ChunkSize  int64  `json:"chunk_size"`
}

// UploadSession represents an active upload session
type UploadSession struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	FileID         *uuid.UUID
	Filename       string
	SizeBytes      int64
	EncryptionKey  []byte
	ChunkCount     int
	ReceivedChunks int
	Status         string
	ExpiresAt      time.Time
}

// UploadService handles file upload operations
type UploadService struct {
	db        *storage.DB
	chunkSize int64
	replicas  int
}

// NewUploadService creates a new upload service
func NewUploadService(db *storage.DB, chunkSize int64, replicas int) *UploadService {
	return &UploadService{
		db:        db,
		chunkSize: chunkSize,
		replicas:  replicas,
	}
}

// InitiateUpload creates a new upload session
func (s *UploadService) InitiateUpload(ctx context.Context, userID uuid.UUID, req InitiateUploadRequest) (*UploadSession, error) {
	// Generate encryption key (256-bit)
	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Calculate chunk count
	chunkCount := int(math.Ceil(float64(req.SizeBytes) / float64(s.chunkSize)))

	session := &UploadSession{
		ID:             uuid.New(),
		UserID:         userID,
		Filename:       req.Filename,
		SizeBytes:      req.SizeBytes,
		EncryptionKey:  encryptionKey,
		ChunkCount:     chunkCount,
		ReceivedChunks: 0,
		Status:         "active",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO upload_sessions (id, user_id, filename, size_bytes, encryption_key, chunk_count, received_chunks, status, expires_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		session.ID, session.UserID, session.Filename, session.SizeBytes,
		session.EncryptionKey, session.ChunkCount, session.ReceivedChunks,
		session.Status, session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload session: %w", err)
	}

	return session, nil
}

// GetSession retrieves an upload session
func (s *UploadService) GetSession(ctx context.Context, sessionID uuid.UUID) (*UploadSession, error) {
	var session UploadSession
	var fileID *uuid.UUID
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, user_id, file_id, filename, size_bytes, encryption_key, chunk_count, received_chunks, status, expires_at 
		 FROM upload_sessions WHERE id = $1`,
		sessionID).Scan(
		&session.ID, &session.UserID, &fileID, &session.Filename,
		&session.SizeBytes, &session.EncryptionKey, &session.ChunkCount,
		&session.ReceivedChunks, &session.Status, &session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}
	session.FileID = fileID
	return &session, nil
}

// UpdateSessionStatus updates upload session status
func (s *UploadService) UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, status string) error {
	_, err := s.db.Pool.Exec(ctx,
		"UPDATE upload_sessions SET status = $1 WHERE id = $2",
		status, sessionID)
	return err
}

// UpdateSessionFileID updates the file ID for an upload session
func (s *UploadService) UpdateSessionFileID(ctx context.Context, sessionID uuid.UUID, fileID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx,
		"UPDATE upload_sessions SET file_id = $1 WHERE id = $2",
		fileID, sessionID)
	return err
}
