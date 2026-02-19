package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/federated-storage/coordinator/internal/config"
	"github.com/federated-storage/coordinator/internal/handlers"
	"github.com/federated-storage/coordinator/internal/middleware"
	"github.com/federated-storage/coordinator/internal/p2p"
	"github.com/federated-storage/coordinator/internal/services"
	"github.com/federated-storage/coordinator/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.toml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Warning: failed to load config from %s: %v", configPath, err)
		log.Println("Using default configuration")
		cfg = config.DefaultConfig()
	}

	// Initialize database
	db, err := storage.New(cfg.Database.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		// Get the directory of the executable to find migrations
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			migrationsPath = filepath.Join(execDir, "..", "..", "migrations")
		} else {
			migrationsPath = "./migrations"
		}
	}
	if err := db.Migrate(migrationsPath); err != nil {
		log.Printf("Warning: migrations failed: %v", err)
	}

	// Initialize services
	authService := services.NewAuthService(db)
	nodeService := services.NewNodeService(db)
	fileService := services.NewFileService(db, cfg.Storage.ChunkSizeBytes, cfg.Storage.StorageCreditPerGBMonth)
	chunkService := services.NewChunkService(db, nodeService)
	uploadService := services.NewUploadService(db, cfg.Storage.ChunkSizeBytes, cfg.Storage.DefaultReplicas)
	// Initialize proof service (for background proof challenges)
	_ = services.NewProofService(db, cfg.Storage.ProofDifficulty)

	// Initialize P2P node
	p2pNode, err := p2p.NewNode(cfg.P2P.ListenAddresses, cfg.P2P.EnableTCP, cfg.P2P.EnableQUIC)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}
	defer p2pNode.Close()

	// Start P2P node
	if err := p2pNode.Start(); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}

	log.Printf("P2P node started with ID: %s", p2pNode.Host().ID().String())

	// Set up HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Serve Web UI static files
	router.Static("/web", "./web/static")
	router.StaticFile("/", "./web/static/index.html")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, os.Getenv("JWT_SECRET"))
	nodeHandler := handlers.NewNodeHandler(nodeService)
	fileHandler := handlers.NewFileHandler(fileService, chunkService)
	uploadHandler := handlers.NewUploadHandler(uploadService, fileService, chunkService, authService, cfg.Storage.DefaultReplicas)

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/credits/purchase", middleware.JWTMiddleware(os.Getenv("JWT_SECRET")), authHandler.PurchaseCredits)
			auth.GET("/profile", middleware.JWTMiddleware(os.Getenv("JWT_SECRET")), authHandler.Profile)
		}

		// Node routes
		nodes := api.Group("/nodes")
		{
			nodes.POST("/register", nodeHandler.Register)
			nodes.GET("", nodeHandler.ListNodes)
			nodes.POST("/heartbeat", middleware.NodeAuthMiddleware(nodeService.GetAPIKeyHash), nodeHandler.Heartbeat)
			nodes.GET("/balance", middleware.NodeAuthMiddleware(nodeService.GetAPIKeyHash), nodeHandler.GetBalance)
		}

		// File routes (protected)
		files := api.Group("/files")
		files.Use(middleware.JWTMiddleware(os.Getenv("JWT_SECRET")))
		{
			files.GET("", fileHandler.ListFiles)
			files.GET("/:id/download", fileHandler.DownloadFile)
			files.DELETE("/:id", fileHandler.DeleteFile)
			files.POST("/upload/initiate", uploadHandler.InitiateUpload)
			files.POST("/upload/:id/chunk", uploadHandler.UploadChunk)
			files.POST("/upload/:id/complete", uploadHandler.CompleteUpload)
		}
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("Coordinator HTTP server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server exited")
}
