package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/federated-storage/storage-node/internal/config"
	"github.com/federated-storage/storage-node/internal/p2p"
	"github.com/federated-storage/storage-node/internal/services"
	"github.com/federated-storage/storage-node/internal/storage"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
	db      *storage.DB
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "storage-node",
		Short: "Federated Storage Node - Distributed storage network participant",
		Long:  `A storage node for the Federated Storage Network that stores encrypted file chunks and earns credits.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.toml)")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(chunksCmd())
	rootCmd.AddCommand(drainCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new storage node",
		Long:  `Initialize a new storage node by generating keys and registering with the coordinator.`,
		RunE:  runInit,
	}

	cmd.Flags().String("name", "", "Node name (required)")
	cmd.Flags().String("coordinator-url", "http://localhost:8080", "Coordinator API URL")
	cmd.Flags().Int("max-storage", 100, "Maximum storage in GB")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	coordinatorURL, _ := cmd.Flags().GetString("coordinator-url")
	maxStorage, _ := cmd.Flags().GetInt("max-storage")

	// Create data directory
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create config
	cfg = &config.Config{
		Node: config.NodeConfig{
			Name:         name,
			DataDir:      dataDir,
			MaxStorageGB: maxStorage,
		},
		Coordinator: config.CoordinatorConfig{
			URL: coordinatorURL,
		},
		Storage: config.StorageConfig{
			ChunkDir: filepath.Join(dataDir, "chunks"),
		},
		API: config.APIConfig{
			Host: "127.0.0.1",
			Port: 8090,
		},
	}

	// Ensure directories
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	// Initialize database
	dbPath := filepath.Join(dataDir, "storage.db")
	var err error
	db, err = storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Run migrations
	migrationsPath := "./migrations"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = filepath.Join(os.Getenv("GOPATH"), "src/github.com/federated-storage/storage-node/migrations")
	}
	if err := db.Migrate(migrationsPath); err != nil {
		log.Printf("Warning: migrations failed: %v", err)
	}

	// Generate key pair for P2P
	privKey := make([]byte, 32)
	if _, err := rand.Read(privKey); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	pubKey := make([]byte, 32)
	if _, err := rand.Read(pubKey); err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Initialize P2P node to get peer ID
	p2pNode, err := p2p.NewNode(nil)
	if err != nil {
		return fmt.Errorf("failed to create P2P node: %w", err)
	}
	if err := p2pNode.Start(); err != nil {
		return fmt.Errorf("failed to start P2P node: %w", err)
	}
	peerID := p2pNode.IDString()
	addrs := p2pNode.Addrs()
	p2pNode.Close()

	// Register with coordinator
	coordinatorClient := services.NewCoordinatorClient(&cfg.Coordinator)
	regResp, err := coordinatorClient.RegisterNode(services.RegisterNodeRequest{
		Name:           name,
		PeerID:         peerID,
		PublicKey:      pubKey,
		Address:        addrs[0],
		TotalStorageGB: maxStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to register with coordinator: %w", err)
	}

	// Save config with API key
	cfg.Coordinator.PeerID = peerID
	cfg.Coordinator.APIKey = regResp.APIKey

	// Save private key
	keyFile := filepath.Join(dataDir, "private.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(privKey)), 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save config
	configPath := "config.toml"
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Storage node initialized successfully!\n")
	fmt.Printf("Node ID: %s\n", regResp.NodeID)
	fmt.Printf("Peer ID: %s\n", peerID)
	fmt.Printf("API Key: %s\n", regResp.APIKey)
	fmt.Printf("Config saved to: %s\n", configPath)

	return nil
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the storage node",
		Long:  `Start the storage node and begin participating in the storage network.`,
		RunE:  runStart,
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load config
	if cfgFile == "" {
		cfgFile = "config.toml"
	}

	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database
	dbPath := filepath.Join(cfg.Node.DataDir, "storage.db")
	db, err = storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize services
	chunkService := services.NewChunkService(db, cfg.Storage.ChunkDir)
	coordinatorClient := services.NewCoordinatorClient(&cfg.Coordinator)
	proofEngine := services.NewProofEngine(chunkService)

	// Initialize P2P node
	p2pNode, err := p2p.NewNode(cfg.P2P.ListenAddresses)
	if err != nil {
		return fmt.Errorf("failed to create P2P node: %w", err)
	}

	// Start P2P node first (this creates the host)
	if err := p2pNode.Start(); err != nil {
		return fmt.Errorf("failed to start P2P node: %w", err)
	}
	defer p2pNode.Close()

	// Set up P2P handlers (must be after Start())
	p2pNode.SetChunkStoreHandler(func(chunkID string, data []byte) error {
		log.Printf("Storing chunk: %s", chunkID)
		// In full implementation, validate hash and store data
		return nil
	})

	p2pNode.SetChunkRetrieveHandler(func(chunkID string) ([]byte, error) {
		log.Printf("Retrieving chunk: %s", chunkID)
		// In full implementation, read from disk
		return []byte{}, nil
	})

	p2pNode.SetProofChallengeHandler(func(chunkID string, seed []byte, difficulty int) (string, int64, error) {
		log.Printf("Processing proof challenge for chunk: %s", chunkID)
		result, err := proofEngine.GenerateProof(chunkID, seed, difficulty)
		if err != nil {
			return "", 0, err
		}
		return result.ProofHash, result.DurationMs, nil
	})

	log.Printf("Storage node started with Peer ID: %s", p2pNode.IDString())
	log.Printf("Listening on:")
	for _, addr := range p2pNode.Addrs() {
		log.Printf("  %s", addr)
	}

	// Start heartbeat loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				totalStorage, _ := chunkService.GetTotalStorage()
				resp, err := coordinatorClient.SendHeartbeat(totalStorage)
				if err != nil {
					log.Printf("Heartbeat failed: %v", err)
				} else {
					log.Printf("Heartbeat sent. Earned credits: %d", resp.EarnedCredits)
				}
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down storage node...")
	return nil
}

func chunksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chunks",
		Short: "Manage stored chunks",
		Long:  `List and manage chunks stored on this node.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stored chunks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfgFile == "" {
				cfgFile = "config.toml"
			}

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			dbPath := filepath.Join(cfg.Node.DataDir, "storage.db")
			db, err := storage.New(dbPath)
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			defer db.Close()

			chunkService := services.NewChunkService(db, cfg.Storage.ChunkDir)
			chunks, err := chunkService.ListChunks()
			if err != nil {
				return fmt.Errorf("failed to list chunks: %w", err)
			}

			count, _ := chunkService.GetChunkCount()
			total, _ := chunkService.GetTotalStorage()

			fmt.Printf("Stored Chunks (%d total, %d bytes used):\n", count, total)
			fmt.Printf("%-64s %-36s %-10s %-12s\n", "CHUNK ID", "FILE ID", "INDEX", "SIZE")
			fmt.Println(string(make([]byte, 126)))
			for _, chunk := range chunks {
				fmt.Printf("%-64s %-36s %-10d %-12d\n", chunk.ID, chunk.FileID, chunk.ChunkIndex, chunk.SizeBytes)
			}

			return nil
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func drainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "drain",
		Short: "Drain the node (stop accepting new chunks)",
		Long:  `Put the node in drain mode - it will finish active operations but reject new chunks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Node set to drain mode. It will finish active operations but reject new chunks.")
			fmt.Println("Note: In MVP, this is a placeholder. Use Ctrl+C to stop the node gracefully.")
			return nil
		},
	}
}
