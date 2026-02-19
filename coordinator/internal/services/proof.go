package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/google/uuid"
)

// ProofService handles proof-of-storage operations
type ProofService struct {
	db         *storage.DB
	difficulty int
}

// NewProofService creates a new proof service
func NewProofService(db *storage.DB, difficulty int) *ProofService {
	return &ProofService{
		db:         db,
		difficulty: difficulty,
	}
}

// CreateChallenge creates a new proof challenge for a chunk
func (s *ProofService) CreateChallenge(ctx context.Context, chunkID, nodeID uuid.UUID) (*models.ProofChallenge, error) {
	// Generate random seed
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("failed to generate seed: %w", err)
	}

	challenge := &models.ProofChallenge{
		ID:         uuid.New(),
		ChunkID:    chunkID,
		NodeID:     nodeID,
		Seed:       seed,
		Difficulty: s.difficulty,
		Status:     "pending",
	}

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO proof_challenges (id, chunk_id, node_id, seed, difficulty, status) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		challenge.ID, challenge.ChunkID, challenge.NodeID, challenge.Seed, challenge.Difficulty, challenge.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge: %w", err)
	}

	return challenge, nil
}

// GetPendingChallenges retrieves pending challenges for a node
func (s *ProofService) GetPendingChallenges(ctx context.Context, nodeID uuid.UUID) ([]models.ProofChallenge, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, chunk_id, node_id, seed, difficulty, status, created_at 
		 FROM proof_challenges 
		 WHERE node_id = $1 AND status = 'pending'`,
		nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []models.ProofChallenge
	for rows.Next() {
		var c models.ProofChallenge
		err := rows.Scan(&c.ID, &c.ChunkID, &c.NodeID, &c.Seed, &c.Difficulty, &c.Status, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}
	return challenges, nil
}

// VerifyProof verifies a proof response from a storage node
func (s *ProofService) VerifyProof(ctx context.Context, challengeID uuid.UUID, proofHash string, durationMs int) error {
	// Get challenge
	var challenge models.ProofChallenge
	err := s.db.Pool.QueryRow(ctx,
		"SELECT id, chunk_id, node_id, seed, difficulty FROM proof_challenges WHERE id = $1",
		challengeID).Scan(&challenge.ID, &challenge.ChunkID, &challenge.NodeID, &challenge.Seed, &challenge.Difficulty)
	if err != nil {
		return fmt.Errorf("challenge not found")
	}

	// Verify timing (should complete within 2 seconds)
	if durationMs > 2000 {
		// Mark as failed due to timeout
		_, err = s.db.Pool.Exec(ctx,
			"UPDATE proof_challenges SET status = 'failed', duration_ms = $1, verified_at = $2 WHERE id = $3",
			durationMs, time.Now(), challengeID)
		return fmt.Errorf("proof verification timed out")
	}

	// Verify proof hash (simplified - in production would verify against actual chunk data)
	expectedHash := s.generateExpectedProof(challenge.Seed, challenge.ChunkID.String())
	if proofHash != expectedHash {
		// Mark as failed
		_, err = s.db.Pool.Exec(ctx,
			"UPDATE proof_challenges SET status = 'failed', proof_hash = $1, duration_ms = $2, verified_at = $3 WHERE id = $4",
			proofHash, durationMs, time.Now(), challengeID)
		return fmt.Errorf("invalid proof hash")
	}

	// Mark as verified
	_, err = s.db.Pool.Exec(ctx,
		`UPDATE proof_challenges 
		 SET status = 'verified', proof_hash = $1, duration_ms = $2, verified_at = $3 
		 WHERE id = $4`,
		proofHash, durationMs, time.Now(), challengeID)
	if err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}

	return nil
}

// GetNodeProofStats retrieves proof statistics for a node
func (s *ProofService) GetNodeProofStats(ctx context.Context, nodeID uuid.UUID, since time.Time) (verified, failed, total int, avgDurationMs float64, err error) {
	err = s.db.Pool.QueryRow(ctx,
		`SELECT 
			COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(*) as total,
			COALESCE(AVG(duration_ms), 0) as avg_duration
		 FROM proof_challenges 
		 WHERE node_id = $1 AND created_at >= $2`,
		nodeID, since).Scan(&verified, &failed, &total, &avgDurationMs)
	return
}

// generateExpectedProof generates the expected proof hash (deterministic)
func (s *ProofService) generateExpectedProof(seed []byte, chunkID string) string {
	// In a real implementation, this would use the actual chunk data
	// For MVP, we create a deterministic hash from seed and chunk ID
	data := append(seed, []byte(chunkID)...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GenerateProofChallengeData generates proof challenge for sending to node
type ProofChallengeData struct {
	ChallengeID string `json:"challenge_id"`
	ChunkID     string `json:"chunk_id"`
	Seed        []byte `json:"seed"`
	Difficulty  int    `json:"difficulty"`
}

// GetChallengesNeedingVerification retrieves challenges that need to be sent to nodes
func (s *ProofService) GetChallengesNeedingVerification(ctx context.Context, limit int) ([]ProofChallengeData, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT pc.id, pc.chunk_id, pc.seed, pc.difficulty, sn.peer_id
		 FROM proof_challenges pc
		 JOIN storage_nodes sn ON pc.node_id = sn.id
		 WHERE pc.status = 'pending'
		 LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []ProofChallengeData
	for rows.Next() {
		var c ProofChallengeData
		var peerID string
		err := rows.Scan(&c.ChallengeID, &c.ChunkID, &c.Seed, &c.Difficulty, &peerID)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}
	return challenges, nil
}
