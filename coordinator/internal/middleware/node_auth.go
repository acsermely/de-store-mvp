package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NodeAuthMiddleware creates middleware for node API key authentication
func NodeAuthMiddleware(getAPIKeyHash func(peerID string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		peerID := c.GetHeader("X-Peer-ID")
		apiKey := c.GetHeader("X-API-Key")

		if peerID == "" || apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing credentials"})
			c.Abort()
			return
		}

		expectedHash, err := getAPIKeyHash(peerID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedHash)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		c.Set("peer_id", peerID)
		c.Next()
	}
}
