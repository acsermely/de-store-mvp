package handlers

import (
	"encoding/base64"
	"net/http"

	"github.com/federated-storage/coordinator/internal/middleware"
	"github.com/federated-storage/coordinator/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	uploadService *services.UploadService
	fileService   *services.FileService
	chunkService  *services.ChunkService
	authService   *services.AuthService
	replicas      int
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(uploadService *services.UploadService, fileService *services.FileService, chunkService *services.ChunkService, authService *services.AuthService, replicas int) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
		fileService:   fileService,
		chunkService:  chunkService,
		authService:   authService,
		replicas:      replicas,
	}
}

// InitiateUpload handles upload initiation
func (h *UploadHandler) InitiateUpload(c *gin.Context) {
	var req services.InitiateUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Check user credits
	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate required credits
	requiredCredits := h.fileService.CalculateStorageCost(req.SizeBytes, h.replicas)
	if user.Credits < requiredCredits {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":             "insufficient credits",
			"required_credits":  requiredCredits,
			"available_credits": user.Credits,
		})
		return
	}

	session, err := h.uploadService.InitiateUpload(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, services.InitiateUploadResponse{
		SessionID:  session.ID.String(),
		ChunkCount: session.ChunkCount,
		ChunkSize:  256 * 1024, // 256KB
	})
}

// UploadChunkRequest represents a chunk upload request
type UploadChunkRequest struct {
	ChunkIndex int    `json:"chunk_index" binding:"gte=0"`
	Data       string `json:"data"`
}

// UploadChunk handles chunk upload
func (h *UploadHandler) UploadChunk(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req UploadChunkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Get session
	session, err := h.uploadService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Verify ownership
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Select nodes for this chunk
	nodes, err := h.chunkService.SelectNodesForChunks(c.Request.Context(), h.replicas)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// Extract node IDs
	nodeIDs := make([]uuid.UUID, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}

	// Create file record if first chunk
	var fileID uuid.UUID
	if session.FileID == nil {
		file, err := h.fileService.CreateFile(c.Request.Context(), userID, session.Filename, session.SizeBytes, "", session.EncryptionKey, session.ChunkCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fileID = file.ID
		err = h.uploadService.UpdateSessionFileID(c.Request.Context(), sessionID, fileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		fileID = *session.FileID
	}

	// Decode base64 data from frontend
	chunkData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 data"})
		return
	}

	// Encrypt chunk
	encryptedData, err := services.EncryptChunk(chunkData, session.EncryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	// Store chunk
	_, err = h.chunkService.StoreChunk(c.Request.Context(), fileID, req.ChunkIndex, encryptedData, nodeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chunk_index": req.ChunkIndex,
		"status":      "stored",
	})
}

// CompleteUpload handles upload completion
func (h *UploadHandler) CompleteUpload(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	session, err := h.uploadService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Deduct credits
	requiredCredits := h.fileService.CalculateStorageCost(session.SizeBytes, h.replicas)
	err = h.authService.UpdateCredits(c.Request.Context(), userID, -requiredCredits, "Storage payment for "+session.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update session status
	err = h.uploadService.UpdateSessionStatus(c.Request.Context(), sessionID, "completed")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if session.FileID != nil {
		err = h.fileService.MarkFileComplete(c.Request.Context(), *session.FileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "completed",
		"credits_deducted": requiredCredits,
	})
}
