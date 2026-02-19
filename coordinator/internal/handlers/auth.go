package handlers

import (
	"net/http"
	"time"

	"github.com/federated-storage/coordinator/internal/middleware"
	"github.com/federated-storage/coordinator/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *services.AuthService
	jwtConfig   middleware.JWTConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtConfig: middleware.JWTConfig{
			Secret:     jwtSecret,
			Expiration: 24 * time.Hour,
		},
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate token
	token, err := middleware.GenerateToken(user.ID.String(), user.Email, h.jwtConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, services.AuthResponse{
		UserID: user.ID.String(),
		Email:  user.Email,
		Token:  token,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Generate token
	token, err := middleware.GenerateToken(user.ID.String(), user.Email, h.jwtConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, services.AuthResponse{
		UserID: user.ID.String(),
		Email:  user.Email,
		Token:  token,
	})
}

// Profile handles getting user profile
func (h *AuthHandler) Profile(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// PurchaseCreditsRequest represents a credit purchase request
type PurchaseCreditsRequest struct {
	AmountUSD int `json:"amount_usd" binding:"required,min=1"`
}

// PurchaseCredits handles credit purchase (mock for MVP)
func (h *AuthHandler) PurchaseCredits(c *gin.Context) {
	var req PurchaseCreditsRequest
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

	// Mock conversion: $1 = 1000 credits
	credits := int64(req.AmountUSD * 1000)

	err = h.authService.UpdateCredits(c.Request.Context(), userID, credits, "Credit purchase")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount_usd":    req.AmountUSD,
		"credits_added": credits,
	})
}
