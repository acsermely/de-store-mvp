package handlers

import (
	"net/http"

	"github.com/federated-storage/coordinator/internal/services"
	"github.com/gin-gonic/gin"
)

// NodeHandler handles storage node requests
type NodeHandler struct {
	nodeService *services.NodeService
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(nodeService *services.NodeService) *NodeHandler {
	return &NodeHandler{nodeService: nodeService}
}

// Register handles node registration
func (h *NodeHandler) Register(c *gin.Context) {
	var req services.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, apiKey, err := h.nodeService.RegisterNode(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, services.RegisterNodeResponse{
		NodeID: node.ID.String(),
		APIKey: apiKey,
	})
}

// ListNodes handles listing all storage nodes
func (h *NodeHandler) ListNodes(c *gin.Context) {
	nodes, err := h.nodeService.GetAllNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	UsedStorageBytes int64 `json:"used_storage_bytes"`
}

// Heartbeat handles node heartbeat
func (h *NodeHandler) Heartbeat(c *gin.Context) {
	peerID := c.GetHeader("X-Peer-ID")
	if peerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing peer id"})
		return
	}

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := h.nodeService.GetNodeByPeerID(c.Request.Context(), peerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	err = h.nodeService.UpdateHeartbeat(c.Request.Context(), node.ID, req.UsedStorageBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"earned_credits": node.EarnedCredits,
	})
}

// GetBalance handles getting node balance/earnings
func (h *NodeHandler) GetBalance(c *gin.Context) {
	peerID := c.GetHeader("X-Peer-ID")
	if peerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing peer id"})
		return
	}

	node, err := h.nodeService.GetNodeByPeerID(c.Request.Context(), peerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id":            node.ID,
		"earned_credits":     node.EarnedCredits,
		"used_storage_bytes": node.UsedStorageBytes,
		"uptime_percentage":  node.UptimePercentage,
	})
}
