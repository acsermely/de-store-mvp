package services

import (
	"context"
	"fmt"
	"time"

	"github.com/federated-storage/coordinator/internal/models"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/google/uuid"
)

// NodeService handles storage node operations
type NodeService struct {
	db *storage.DB
}

// NewNodeService creates a new node service
func NewNodeService(db *storage.DB) *NodeService {
	return &NodeService{db: db}
}

// RegisterNodeRequest represents a node registration request
type RegisterNodeRequest struct {
	Name           string `json:"name" binding:"required"`
	PeerID         string `json:"peer_id" binding:"required"`
	PublicKey      []byte `json:"public_key" binding:"required"`
	Address        string `json:"address"`
	TotalStorageGB int    `json:"total_storage_gb"`
}

// RegisterNodeResponse represents a node registration response
type RegisterNodeResponse struct {
	NodeID string `json:"node_id"`
	APIKey string `json:"api_key"`
}

// RegisterNode registers a new storage node
func (s *NodeService) RegisterNode(ctx context.Context, req RegisterNodeRequest) (*models.StorageNode, string, error) {
	// Check if peer ID already exists
	var exists bool
	err := s.db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM storage_nodes WHERE peer_id = $1)",
		req.PeerID).Scan(&exists)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check node existence: %w", err)
	}
	if exists {
		return nil, "", fmt.Errorf("node with this peer_id already exists")
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate api key: %w", err)
	}

	// Hash the API key for storage
	apiKeyHash := hashAPIKey(apiKey)

	node := &models.StorageNode{
		ID:                uuid.New(),
		Name:              req.Name,
		PeerID:            req.PeerID,
		PublicKey:         req.PublicKey,
		Address:           req.Address,
		APIKeyHash:        apiKeyHash,
		Status:            "active",
		TotalStorageBytes: int64(req.TotalStorageGB) * 1024 * 1024 * 1024,
		UsedStorageBytes:  0,
		EarnedCredits:     0,
		UptimePercentage:  100.0,
		LastHeartbeat:     nil,
	}

	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO storage_nodes (id, name, peer_id, public_key, address, api_key_hash, status, total_storage_bytes, used_storage_bytes, earned_credits) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		node.ID, node.Name, node.PeerID, node.PublicKey, node.Address,
		node.APIKeyHash, node.Status, node.TotalStorageBytes, node.UsedStorageBytes, node.EarnedCredits)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create node: %w", err)
	}

	return node, apiKey, nil
}

// GetNodeByPeerID retrieves a node by peer ID
func (s *NodeService) GetNodeByPeerID(ctx context.Context, peerID string) (*models.StorageNode, error) {
	var node models.StorageNode
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, name, peer_id, public_key, address, api_key_hash, status, total_storage_bytes, 
		 used_storage_bytes, earned_credits, uptime_percentage, last_heartbeat, created_at, updated_at 
		 FROM storage_nodes WHERE peer_id = $1`,
		peerID).Scan(
		&node.ID, &node.Name, &node.PeerID, &node.PublicKey, &node.Address,
		&node.APIKeyHash, &node.Status, &node.TotalStorageBytes, &node.UsedStorageBytes,
		&node.EarnedCredits, &node.UptimePercentage, &node.LastHeartbeat,
		&node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("node not found")
	}
	return &node, nil
}

// GetAllNodes retrieves all active storage nodes
func (s *NodeService) GetAllNodes(ctx context.Context) ([]models.StorageNode, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, name, peer_id, public_key, address, status, total_storage_bytes, 
		 used_storage_bytes, earned_credits, uptime_percentage, last_heartbeat, created_at 
		 FROM storage_nodes WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []models.StorageNode
	for rows.Next() {
		var node models.StorageNode
		err := rows.Scan(
			&node.ID, &node.Name, &node.PeerID, &node.PublicKey, &node.Address,
			&node.Status, &node.TotalStorageBytes, &node.UsedStorageBytes,
			&node.EarnedCredits, &node.UptimePercentage, &node.LastHeartbeat,
			&node.CreatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// UpdateHeartbeat updates node heartbeat
func (s *NodeService) UpdateHeartbeat(ctx context.Context, nodeID uuid.UUID, usedBytes int64) error {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE storage_nodes 
		 SET last_heartbeat = $1, used_storage_bytes = $2, updated_at = $3 
		 WHERE id = $4`,
		now, usedBytes, now, nodeID)
	return err
}

// GetAPIKeyHash retrieves the API key hash for a peer ID (for middleware)
func (s *NodeService) GetAPIKeyHash(peerID string) (string, error) {
	var hash string
	err := s.db.Pool.QueryRow(context.Background(),
		"SELECT api_key_hash FROM storage_nodes WHERE peer_id = $1 AND status = 'active'",
		peerID).Scan(&hash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// Helper functions
func generateAPIKey() (string, error) {
	// Generate a random API key (simplified for MVP)
	// In production, use crypto/rand for better security
	return fmt.Sprintf("fsn_%s", uuid.New().String()), nil
}

func hashAPIKey(apiKey string) string {
	// For MVP, we store the API key directly
	// In production, use proper hashing
	return apiKey
}
