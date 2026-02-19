package handlers

import (
	"fmt"
	"net/http"

	"github.com/federated-storage/coordinator/internal/middleware"
	"github.com/federated-storage/coordinator/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileHandler handles file-related requests
type FileHandler struct {
	fileService  *services.FileService
	chunkService *services.ChunkService
}

// NewFileHandler creates a new file handler
func NewFileHandler(fileService *services.FileService, chunkService *services.ChunkService) *FileHandler {
	return &FileHandler{fileService: fileService, chunkService: chunkService}
}

// ListFiles handles listing user files
func (h *FileHandler) ListFiles(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	files, err := h.fileService.GetUserFiles(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// DownloadFile handles file download
func (h *FileHandler) DownloadFile(c *gin.Context) {
	fileIDStr := c.Param("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	file, err := h.fileService.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if file.Status != "ready" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not ready"})
		return
	}

	chunks, err := h.chunkService.GetChunksByFileWithData(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve chunks"})
		return
	}

	var decryptedData []byte
	for i := 0; i < file.ChunkCount; i++ {
		chunkData, ok := chunks[i]
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("missing chunk %d", i)})
			return
		}

		decrypted, err := services.DecryptChunk(chunkData, file.EncryptionKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to decrypt chunk %d", i)})
			return
		}
		decryptedData = append(decryptedData, decrypted...)
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(decryptedData)))
	c.Data(http.StatusOK, "application/octet-stream", decryptedData)
}

// DeleteFile handles file deletion
func (h *FileHandler) DeleteFile(c *gin.Context) {
	fileIDStr := c.Param("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	file, err := h.fileService.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	err = h.fileService.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
