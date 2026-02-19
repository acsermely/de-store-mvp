# Implementation Plan: Missing Features

## Executive Summary

This document outlines the detailed implementation plan for completing the Federated Storage Network MVP. The system currently has a solid foundation with HTTP API, authentication, encryption, and basic P2P setup, but lacks actual chunk distribution, proof scheduling, node economics, and reliability features.

**Target Scale:** 10-50 nodes (medium production deployment)  
**Data Durability:** 3 replicas minimum  
**NAT Support:** Yes, with relay nodes  
**Priority:** P2P chunk distribution first (most critical)

---

## Current System Status

### ✅ Implemented
- Database schemas (PostgreSQL + SQLite)
- JWT authentication with bcrypt
- AES-256-GCM encryption
- HTTP API endpoints
- Basic libp2p with DHT
- Credit system (mock purchases)
- File upload/download flow (within coordinator)

### ⚠️ Partially Implemented
- P2P protocol layer (handlers exist but are placeholders)
- Node registration and heartbeat
- Proof challenge data structures (no scheduler running)

### ❌ Missing
- Actual chunk distribution to storage nodes
- Chunk retrieval from storage nodes
- Automatic re-replication
- Proof challenge scheduler
- Node earnings calculation
- File deletion propagation
- NAT traversal with relays

---

## Phase 1: P2P Protocol & Chunk Distribution (Weeks 1-2)

### 1.1 Protocol Buffers Schema

**New file**: `coordinator/proto/storage.proto`

```protobuf
syntax = "proto3";
package federated_storage;

message StoreChunkRequest {
    string chunk_id = 1;
    string file_id = 2;
    int32 chunk_index = 3;
    string hash = 4;
    int32 size_bytes = 5;
    bytes data = 6;
}

message StoreChunkResponse {
    bool success = 1;
    string error = 2;
}

message RetrieveChunkRequest {
    string chunk_id = 1;
}

message RetrieveChunkResponse {
    bytes data = 1;
    string hash = 2;
    bool found = 3;
}

message DeleteChunkRequest {
    string chunk_id = 1;
}

message DeleteChunkResponse {
    bool success = 1;
}

message ProofChallengeRequest {
    string challenge_id = 1;
    string chunk_id = 2;
    bytes seed = 3;
    int32 difficulty = 4;
}

message ProofChallengeResponse {
    string challenge_id = 1;
    string proof_hash = 2;
    int64 duration_ms = 3;
}
```

### 1.2 Enhanced P2P Node (Coordinator)

**File**: `coordinator/internal/p2p/node.go`

Add the following methods:

```go
// SendChunkToNode sends a chunk to a storage node via P2P
func (n *Node) SendChunkToNode(ctx context.Context, peerID string, req *pb.StoreChunkRequest) error

// RetrieveChunkFromNode retrieves a chunk from a storage node
func (n *Node) RetrieveChunkFromNode(ctx context.Context, peerID string, chunkID string) (*pb.RetrieveChunkResponse, error)

// SendDeleteCommand sends delete command to a storage node
func (n *Node) SendDeleteCommand(ctx context.Context, peerID string, chunkID string) error

// SendProofChallenge sends a proof challenge to a storage node
func (n *Node) SendProofChallenge(ctx context.Context, peerID string, req *pb.ProofChallengeRequest) (*pb.ProofChallengeResponse, error)
```

**Implementation details:**
- Use length-prefixed message encoding for stream reliability
- Implement retry logic with exponential backoff (max 3 retries)
- Support concurrent transfers (max 5 parallel per node)
- Set transfer timeout to 60 seconds per chunk

### 1.3 Enhanced P2P Node (Storage Node)

**File**: `storage-node/internal/p2p/node.go`

Replace placeholder handlers with full implementations:

```go
// SetChunkStoreHandler - full implementation
func (n *Node) SetChunkStoreHandler(handler func(*pb.StoreChunkRequest) (*pb.StoreChunkResponse, error)) {
    n.host.SetStreamHandler("/federated-storage/1.0.0/store-chunk", func(s network.Stream) {
        defer s.Close()
        
        // Read length-prefixed message
        req, err := readStoreChunkRequest(s)
        if err != nil {
            sendErrorResponse(s, err)
            return
        }
        
        // Call handler
        resp, err := handler(req)
        if err != nil {
            sendErrorResponse(s, err)
            return
        }
        
        // Send response
        writeStoreChunkResponse(s, resp)
    })
}

// SetChunkRetrieveHandler - full implementation
func (n *Node) SetChunkRetrieveHandler(handler func(*pb.RetrieveChunkRequest) (*pb.RetrieveChunkResponse, error))

// SetProofChallengeHandler - full implementation
func (n *Node) SetProofChallengeHandler(handler func(*pb.ProofChallengeRequest) (*pb.ProofChallengeResponse, error))
```

### 1.4 Storage Node Chunk Handlers

**Modify**: `storage-node/cmd/storage-node/main.go`

Replace placeholder handlers:

```go
// Chunk store handler - actually stores data
p2pNode.SetChunkStoreHandler(func(req *pb.StoreChunkRequest) (*pb.StoreChunkResponse, error) {
    // 1. Validate hash
    calculatedHash := sha256.Sum256(req.Data)
    if hex.EncodeToString(calculatedHash[:]) != req.Hash {
        return &pb.StoreChunkResponse{Success: false, Error: "hash mismatch"}, nil
    }
    
    // 2. Store on disk
    err := chunkService.StoreChunk(req.ChunkId, req.FileId, int(req.ChunkIndex), req.Hash, req.Data)
    if err != nil {
        return &pb.StoreChunkResponse{Success: false, Error: err.Error()}, nil
    }
    
    return &pb.StoreChunkResponse{Success: true}, nil
})

// Chunk retrieve handler
p2pNode.SetChunkRetrieveHandler(func(req *pb.RetrieveChunkRequest) (*pb.RetrieveChunkResponse, error) {
    data, err := chunkService.GetChunkData(req.ChunkId)
    if err != nil {
        return &pb.RetrieveChunkResponse{Found: false}, nil
    }
    
    // Calculate hash
    hash := sha256.Sum256(data)
    
    return &pb.RetrieveChunkResponse{
        Data: data,
        Hash: hex.EncodeToString(hash[:]),
        Found: true,
    }, nil
})

// Proof challenge handler
p2pNode.SetProofChallengeHandler(func(req *pb.ProofChallengeRequest) (*pb.ProofChallengeResponse, error) {
    result, err := proofEngine.GenerateProof(req.ChunkId, req.Seed, int(req.Difficulty))
    if err != nil {
        return nil, err
    }
    
    return &pb.ProofChallengeResponse{
        ChallengeId: req.ChallengeId,
        ProofHash: result.ProofHash,
        DurationMs: result.DurationMs,
    }, nil
})
```

### 1.5 Update Upload Handler

**File**: `coordinator/internal/handlers/upload.go`

**Modify `UploadChunk` method**:

```go
func (h *UploadHandler) UploadChunk(c *gin.Context) {
    // ... existing validation code ...
    
    // Select nodes for this chunk
    nodes, err := h.chunkService.SelectNodesForChunks(c.Request.Context(), h.replicas)
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
        return
    }
    
    // Create file record if first chunk
    var fileID uuid.UUID
    if session.FileID == nil {
        file, err := h.fileService.CreateFile(c.Request.Context(), userID, 
            session.Filename, session.SizeBytes, "", session.EncryptionKey, session.ChunkCount)
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
    
    // Calculate hash
    hash := sha256.Sum256(encryptedData)
    hashStr := hex.EncodeToString(hash[:])
    
    // Store chunk metadata in DB
    chunk, err := h.chunkService.StoreChunk(c.Request.Context(), fileID, req.ChunkIndex, encryptedData, nodeIDs)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // Distribute to nodes via P2P (async)
    var wg sync.WaitGroup
    errChan := make(chan error, len(nodes))
    
    for _, node := range nodes {
        wg.Add(1)
        go func(n models.StorageNode) {
            defer wg.Done()
            
            req := &pb.StoreChunkRequest{
                ChunkId:    chunk.ID.String(),
                FileId:     fileID.String(),
                ChunkIndex: int32(req.ChunkIndex),
                Hash:       hashStr,
                SizeBytes:  int32(len(encryptedData)),
                Data:       encryptedData,
            }
            
            // Retry logic
            var lastErr error
            for i := 0; i < 3; i++ {
                ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
                lastErr = h.p2pNode.SendChunkToNode(ctx, n.PeerID, req)
                cancel()
                
                if lastErr == nil {
                    // Update assignment status to active
                    h.chunkService.UpdateAssignmentStatus(c.Request.Context(), chunk.ID, n.ID, "active")
                    return
                }
                
                time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
            }
            
            // Mark as failed after retries
            h.chunkService.UpdateAssignmentStatus(c.Request.Context(), chunk.ID, n.ID, "failed")
            errChan <- fmt.Errorf("failed to send chunk to node %s: %w", n.PeerID, lastErr)
        }(node)
    }
    
    // Wait for all transfers (with timeout)
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        // Check if at least 1 node succeeded
        successCount := h.chunkService.CountActiveAssignments(c.Request.Context(), chunk.ID)
        if successCount == 0 {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to distribute chunk to any node"})
            return
        }
        
        // If less than required replicas, trigger async replication
        if successCount < h.replicas {
            go h.replicationService.ReplicateChunk(c.Request.Context(), chunk.ID, h.replicas)
        }
        
    case <-time.After(120 * time.Second):
        // Timeout - chunk metadata stored, will be replicated later
    }
    
    c.JSON(http.StatusOK, gin.H{
        "chunk_index": req.ChunkIndex,
        "status":      "stored",
        "nodes":       len(nodes),
    })
}
```

### 1.6 Update Download Handler

**File**: `coordinator/internal/handlers/file.go`

**Modify `DownloadFile` method**:

```go
func (h *FileHandler) DownloadFile(c *gin.Context) {
    // ... existing validation code ...
    
    // Get chunk assignments with active nodes
    assignments, err := h.chunkService.GetChunksByFileWithAssignments(c.Request.Context(), fileID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve chunk assignments"})
        return
    }
    
    var decryptedData []byte
    for i := 0; i < file.ChunkCount; i++ {
        chunkAssignments, ok := assignments[i]
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("missing chunk %d", i)})
            return
        }
        
        // Try to fetch from nodes
        var chunkData []byte
        var fetchErr error
        
        for _, assignment := range chunkAssignments {
            if assignment.NodeStatus != "active" {
                continue
            }
            
            ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
            resp, err := h.p2pNode.RetrieveChunkFromNode(ctx, assignment.PeerID, assignment.ChunkID.String())
            cancel()
            
            if err == nil && resp.Found {
                chunkData = resp.Data
                break
            }
        }
        
        // If failed to fetch from nodes, try coordinator's backup
        if chunkData == nil {
            chunkData, err = h.chunkService.GetChunkData(c.Request.Context(), chunkAssignments[0].ChunkID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to retrieve chunk %d from any source", i)})
                return
            }
        }
        
        // Decrypt
        decrypted, err := services.DecryptChunk(chunkData, file.EncryptionKey)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to decrypt chunk %d", i)})
            return
        }
        decryptedData = append(decryptedData, decrypted...)
    }
    
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Filename))
    c.Header("Content-Type", "application/octet-stream")
    c.Data(http.StatusOK, "application/octet-stream", decryptedData)
}
```

---

## Phase 2: NAT Traversal & Relay Support (Week 3)

### 2.1 Configuration Updates

**File**: `coordinator/internal/config/config.go`

```go
type P2PConfig struct {
    ListenAddresses []string `toml:"listen_addresses"`
    EnableTCP       bool     `toml:"enable_tcp"`
    EnableQUIC      bool     `toml:"enable_quic"`
    EnableRelay     bool     `toml:"enable_relay"`
    RelayAddresses  []string `toml:"relay_addresses"`
}
```

**File**: `storage-node/internal/config/config.go`

```go
type P2PConfig struct {
    ListenAddresses []string `toml:"listen_addresses"`
    BootstrapPeers  []string `toml:"bootstrap_peers"`
    EnableRelay     bool     `toml:"enable_relay"`
    RelayAddresses  []string `toml:"relay_addresses"`
    IsRelayClient   bool     `toml:"is_relay_client"` // true if behind NAT
}
```

### 2.2 Update P2P Node Initialization

**File**: `coordinator/internal/p2p/node.go`

```go
func NewNode(listenAddresses []string, enableTCP, enableQUIC, enableRelay bool, relayAddresses []string) (*Node, error) {
    // Build libp2p options
    opts := []libp2p.Option{
        libp2p.ListenAddrStrings(listenAddresses...),
    }
    
    if enableRelay {
        opts = append(opts, libp2p.EnableRelay())
        
        // Parse relay addresses
        var relays []peer.AddrInfo
        for _, addr := range relayAddresses {
            addrInfo, err := peer.AddrInfoFromString(addr)
            if err != nil {
                return nil, fmt.Errorf("invalid relay address %s: %w", addr, err)
            }
            relays = append(relays, *addrInfo)
        }
        
        if len(relays) > 0 {
            opts = append(opts, libp2p.StaticRelays(relays))
        }
    }
    
    // Create host
    h, err := libp2p.New(opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create libp2p host: %w", err)
    }
    
    // ... rest of initialization
}
```

**File**: `storage-node/internal/p2p/node.go`

```go
func (n *Node) Start() error {
    opts := []libp2p.Option{
        libp2p.ListenAddrStrings(n.config.ListenAddresses...),
    }
    
    if n.config.EnableRelay {
        opts = append(opts, libp2p.EnableRelay())
        
        if n.config.IsRelayClient {
            // Node behind NAT - use relay reservation
            var relays []peer.AddrInfo
            for _, addr := range n.config.RelayAddresses {
                addrInfo, err := peer.AddrInfoFromString(addr)
                if err != nil {
                    continue
                }
                relays = append(relays, *addrInfo)
            }
            
            if len(relays) > 0 {
                opts = append(opts, libp2p.StaticRelays(relays))
            }
        }
    }
    
    // Create host and DHT...
    
    // If behind NAT, reserve relay slots
    if n.config.IsRelayClient && len(relays) > 0 {
        for _, relay := range relays {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            err := n.host.Connect(ctx, relay)
            cancel()
            if err != nil {
                log.Printf("Failed to connect to relay %s: %v", relay.ID, err)
            }
        }
    }
}

// GetRelayAddresses returns relay addresses for this node
func (n *Node) GetRelayAddresses() []string {
    var addrs []string
    for _, addr := range n.host.Addrs() {
        if strings.Contains(addr.String(), "/p2p-circuit/") {
            addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), n.ID().String()))
        }
    }
    return addrs
}
```

### 2.3 Advertise Relay Addresses

**File**: `storage-node/internal/services/coordinator.go`

```go
func (c *CoordinatorClient) SendHeartbeat(usedBytes int64, relayAddrs []string) (*HeartbeatResponse, error) {
    req := HeartbeatRequest{
        UsedStorageBytes: usedBytes,
        RelayAddresses:   relayAddrs,
    }
    // ... rest of implementation
}
```

**File**: `coordinator/internal/handlers/node.go`

```go
func (h *NodeHandler) Heartbeat(c *gin.Context) {
    // ... existing code ...
    
    // Update relay addresses if provided
    if len(req.RelayAddresses) > 0 {
        h.nodeService.UpdateNodeRelayAddresses(c.Request.Context(), node.ID, req.RelayAddresses)
    }
    
    // ... rest of implementation
}
```

---

## Phase 3: Chunk Replication & Durability (Week 4)

### 3.1 Database Migration

**New file**: `coordinator/migrations/3_replication.up.sql`

```sql
-- Add assignment status tracking
ALTER TABLE chunk_assignments ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'pending';

-- Add relay addresses to storage nodes
ALTER TABLE storage_nodes ADD COLUMN IF NOT EXISTS relay_addresses TEXT[];

-- Add last verified timestamp to chunks
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS last_verified_at TIMESTAMP WITH TIME ZONE;

-- Create index for assignment status
CREATE INDEX IF NOT EXISTS idx_chunk_assignments_status ON chunk_assignments(status);

-- Create index for under-replicated chunks
CREATE INDEX IF NOT EXISTS idx_chunks_verification ON chunks(last_verified_at);
```

### 3.2 Replication Service

**New file**: `coordinator/internal/services/replication.go`

```go
package services

import (
    "context"
    "fmt"
    "time"
    
    "github.com/federated-storage/coordinator/internal/models"
    "github.com/federated-storage/coordinator/internal/p2p"
    "github.com/federated-storage/coordinator/internal/storage"
    "github.com/google/uuid"
)

// ReplicationService handles chunk replication and durability
type ReplicationService struct {
    db           *storage.DB
    p2pNode      *p2p.Node
    chunkService *ChunkService
    nodeService  *NodeService
}

// NewReplicationService creates a new replication service
func NewReplicationService(db *storage.DB, p2pNode *p2p.Node, chunkService *ChunkService, nodeService *NodeService) *ReplicationService {
    return &ReplicationService{
        db:           db,
        p2pNode:      p2pNode,
        chunkService: chunkService,
        nodeService:  nodeService,
    }
}

// UnderReplicatedChunk represents a chunk that needs replication
type UnderReplicatedChunk struct {
    ChunkID       uuid.UUID
    FileID        uuid.UUID
    CurrentCount  int
    TargetCount   int
    Nodes         []models.StorageNode
}

// GetUnderReplicatedChunks finds chunks with fewer than target replicas
func (s *ReplicationService) GetUnderReplicatedChunks(ctx context.Context, targetCount int) ([]UnderReplicatedChunk, error) {
    rows, err := s.db.Pool.Query(ctx, `
        SELECT 
            c.id, c.file_id, c.hash, c.size_bytes,
            COUNT(ca.id) as active_count
        FROM chunks c
        LEFT JOIN chunk_assignments ca ON c.id = ca.chunk_id AND ca.status = 'active'
        GROUP BY c.id, c.file_id, c.hash, c.size_bytes
        HAVING COUNT(ca.id) < $1
    `, targetCount)
    if err != nil {
        return nil, fmt.Errorf("failed to query under-replicated chunks: %w", err)
    }
    defer rows.Close()
    
    var chunks []UnderReplicatedChunk
    for rows.Next() {
        var chunk UnderReplicatedChunk
        var hash string
        var sizeBytes int
        err := rows.Scan(&chunk.ChunkID, &chunk.FileID, &hash, &sizeBytes, &chunk.CurrentCount)
        if err != nil {
            continue
        }
        chunk.TargetCount = targetCount
        chunks = append(chunks, chunk)
    }
    
    return chunks, nil
}

// ReplicateChunk replicates a chunk to reach target replica count
func (s *ReplicationService) ReplicateChunk(ctx context.Context, chunkID uuid.UUID, targetCount int) error {
    // Get chunk info
    var chunk models.Chunk
    err := s.db.Pool.QueryRow(ctx,
        "SELECT id, file_id, hash, size_bytes FROM chunks WHERE id = $1",
        chunkID).Scan(&chunk.ID, &chunk.FileID, &chunk.Hash, &chunk.SizeBytes)
    if err != nil {
        return fmt.Errorf("chunk not found: %w", err)
    }
    
    // Get current active nodes storing this chunk
    currentNodes, err := s.chunkService.GetChunkAssignments(ctx, chunkID)
    if err != nil {
        return fmt.Errorf("failed to get current assignments: %w", err)
    }
    
    neededReplicas := targetCount - len(currentNodes)
    if neededReplicas <= 0 {
        return nil // Already replicated enough
    }
    
    // Get chunk data from an existing healthy node
    var chunkData []byte
    for _, assignment := range currentNodes {
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        resp, err := s.p2pNode.RetrieveChunkFromNode(ctx, assignment.PeerID, chunkID.String())
        cancel()
        
        if err == nil && resp.Found {
            chunkData = resp.Data
            break
        }
    }
    
    if chunkData == nil {
        // Try coordinator backup
        chunkData, err = s.chunkService.GetChunkData(ctx, chunkID)
        if err != nil {
            return fmt.Errorf("failed to retrieve chunk data from any source: %w", err)
        }
    }
    
    // Select new nodes (excluding current)
    allNodes, err := s.nodeService.GetAllNodes(ctx)
    if err != nil {
        return fmt.Errorf("failed to get nodes: %w", err)
    }
    
    // Filter out nodes already storing this chunk
    currentNodeIDs := make(map[uuid.UUID]bool)
    for _, n := range currentNodes {
        currentNodeIDs[n.NodeID] = true
    }
    
    var availableNodes []models.StorageNode
    for _, node := range allNodes {
        if !currentNodeIDs[node.ID] {
            availableNodes = append(availableNodes, node)
        }
    }
    
    if len(availableNodes) < neededReplicas {
        return fmt.Errorf("not enough available nodes (need %d, have %d)", neededReplicas, len(availableNodes))
    }
    
    // Select best nodes (could add more criteria here)
    selectedNodes := availableNodes[:neededReplicas]
    
    // Replicate to selected nodes
    for _, node := range selectedNodes {
        // Create assignment record (pending)
        assignmentID := uuid.New()
        _, err := s.db.Pool.Exec(ctx,
            "INSERT INTO chunk_assignments (id, chunk_id, node_id, status) VALUES ($1, $2, $3, $4)",
            assignmentID, chunkID, node.ID, "pending")
        if err != nil {
            continue
        }
        
        // Send chunk to node
        req := &pb.StoreChunkRequest{
            ChunkId:    chunkID.String(),
            FileId:     chunk.FileID.String(),
            Hash:       chunk.Hash,
            SizeBytes:  int32(chunk.SizeBytes),
            Data:       chunkData,
        }
        
        ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
        err = s.p2pNode.SendChunkToNode(ctx, node.PeerID, req)
        cancel()
        
        if err != nil {
            // Mark as failed
            s.db.Pool.Exec(ctx,
                "UPDATE chunk_assignments SET status = 'failed' WHERE id = $1",
                assignmentID)
            continue
        }
        
        // Mark as active
        _, err = s.db.Pool.Exec(ctx,
            "UPDATE chunk_assignments SET status = 'active' WHERE id = $1",
            assignmentID)
        if err != nil {
            continue
        }
    }
    
    return nil
}

// VerifyChunkReplication checks if a chunk has correct replication
func (s *ReplicationService) VerifyChunkReplication(ctx context.Context, chunkID uuid.UUID, targetCount int) (int, error) {
    assignments, err := s.chunkService.GetChunkAssignments(ctx, chunkID)
    if err != nil {
        return 0, err
    }
    
    // Ping each node to verify they still have the chunk
    var activeCount int
    for _, assignment := range assignments {
        ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
        _, err := s.p2pNode.RetrieveChunkFromNode(ctx, assignment.PeerID, chunkID.String())
        cancel()
        
        if err == nil {
            activeCount++
        } else {
            // Mark assignment as orphaned
            s.db.Pool.Exec(ctx,
                "UPDATE chunk_assignments SET status = 'orphaned' WHERE id = $1",
                assignment.ID)
        }
    }
    
    return activeCount, nil
}
```

### 3.3 Background Replication Job

**New file**: `coordinator/internal/jobs/replication.go`

```go
package jobs

import (
    "context"
    "log"
    "time"
    
    "github.com/federated-storage/coordinator/internal/services"
)

// ReplicationJob periodically checks and fixes under-replicated chunks
type ReplicationJob struct {
    replicationService *services.ReplicationService
    targetReplicas     int
    interval           time.Duration
    stopChan           chan bool
}

// NewReplicationJob creates a new replication job
func NewReplicationJob(service *services.ReplicationService, targetReplicas int, interval time.Duration) *ReplicationJob {
    return &ReplicationJob{
        replicationService: service,
        targetReplicas:     targetReplicas,
        interval:           interval,
        stopChan:           make(chan bool),
    }
}

// Start begins the replication job
func (j *ReplicationJob) Start() {
    log.Printf("Starting replication job (target: %d replicas, interval: %v)", j.targetReplicas, j.interval)
    
    ticker := time.NewTicker(j.interval)
    go func() {
        // Run immediately on start
        j.run()
        
        for {
            select {
            case <-ticker.C:
                j.run()
            case <-j.stopChan:
                ticker.Stop()
                return
            }
        }
    }()
}

// Stop halts the replication job
func (j *ReplicationJob) Stop() {
    close(j.stopChan)
}

func (j *ReplicationJob) run() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    log.Println("Running replication check...")
    
    // Get under-replicated chunks
    chunks, err := j.replicationService.GetUnderReplicatedChunks(ctx, j.targetReplicas)
    if err != nil {
        log.Printf("Failed to get under-replicated chunks: %v", err)
        return
    }
    
    if len(chunks) == 0 {
        return
    }
    
    log.Printf("Found %d under-replicated chunks", len(chunks))
    
    // Process chunks (limit to 100 per run to avoid overload)
    maxChunks := 100
    if len(chunks) > maxChunks {
        chunks = chunks[:maxChunks]
    }
    
    for _, chunk := range chunks {
        log.Printf("Replicating chunk %s (%d/%d replicas)", 
            chunk.ChunkID, chunk.CurrentCount, chunk.TargetCount)
        
        err := j.replicationService.ReplicateChunk(ctx, chunk.ChunkID, j.targetReplicas)
        if err != nil {
            log.Printf("Failed to replicate chunk %s: %v", chunk.ChunkID, err)
        } else {
            log.Printf("Successfully replicated chunk %s", chunk.ChunkID)
        }
        
        // Small delay to avoid overwhelming the network
        time.Sleep(100 * time.Millisecond)
    }
    
    log.Printf("Replication check complete. Processed %d chunks.", len(chunks))
}
```

### 3.4 Update Chunk Service Methods

**File**: `coordinator/internal/services/chunk.go`

Add new methods:

```go
// UpdateAssignmentStatus updates the status of a chunk assignment
func (s *ChunkService) UpdateAssignmentStatus(ctx context.Context, chunkID, nodeID uuid.UUID, status string) error {
    _, err := s.db.Pool.Exec(ctx,
        "UPDATE chunk_assignments SET status = $1 WHERE chunk_id = $2 AND node_id = $3",
        status, chunkID, nodeID)
    return err
}

// CountActiveAssignments counts active assignments for a chunk
func (s *ChunkService) CountActiveAssignments(ctx context.Context, chunkID uuid.UUID) (int, error) {
    var count int
    err := s.db.Pool.QueryRow(ctx,
        "SELECT COUNT(*) FROM chunk_assignments WHERE chunk_id = $1 AND status = 'active'",
        chunkID).Scan(&count)
    return count, err
}

// GetChunkData retrieves chunk data from coordinator's database
func (s *ChunkService) GetChunkData(ctx context.Context, chunkID uuid.UUID) ([]byte, error) {
    var data []byte
    err := s.db.Pool.QueryRow(ctx,
        "SELECT data FROM chunks WHERE id = $1",
        chunkID).Scan(&data)
    if err != nil {
        return nil, fmt.Errorf("chunk data not found: %w", err)
    }
    return data, nil
}

// GetChunksByFileWithAssignments retrieves chunks with their node assignments
func (s *ChunkService) GetChunksByFileWithAssignments(ctx context.Context, fileID uuid.UUID) (map[int][]ChunkAssignmentWithNode, error) {
    rows, err := s.db.Pool.Query(ctx, `
        SELECT 
            c.chunk_index,
            c.id as chunk_id,
            ca.node_id,
            sn.peer_id,
            ca.status as assignment_status,
            sn.status as node_status
        FROM chunks c
        LEFT JOIN chunk_assignments ca ON c.id = ca.chunk_id
        LEFT JOIN storage_nodes sn ON ca.node_id = sn.id
        WHERE c.file_id = $1
        ORDER BY c.chunk_index
    `, fileID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    assignments := make(map[int][]ChunkAssignmentWithNode)
    for rows.Next() {
        var a ChunkAssignmentWithNode
        var chunkIndex int
        err := rows.Scan(&chunkIndex, &a.ChunkID, &a.NodeID, &a.PeerID, &a.AssignmentStatus, &a.NodeStatus)
        if err != nil {
            continue
        }
        assignments[chunkIndex] = append(assignments[chunkIndex], a)
    }
    
    return assignments, nil
}

type ChunkAssignmentWithNode struct {
    ChunkID          uuid.UUID
    NodeID           uuid.UUID
    PeerID           string
    AssignmentStatus string
    NodeStatus       string
}
```

---

## Phase 4: Proof-of-Storage Scheduler (Week 5)

### 4.1 Proof Scheduler Job

**New file**: `coordinator/internal/jobs/proof_scheduler.go`

```go
package jobs

import (
    "context"
    "log"
    "math/rand"
    "time"
    
    "github.com/federated-storage/coordinator/internal/p2p"
    "github.com/federated-storage/coordinator/internal/services"
    "github.com/google/uuid"
)

// ProofScheduler manages proof challenges for storage nodes
type ProofScheduler struct {
    proofService *services.ProofService
    p2pNode      *p2p.Node
    nodeService  *services.NodeService
    interval     time.Duration
    stopChan     chan bool
    difficulty   int
}

// NewProofScheduler creates a new proof scheduler
func NewProofScheduler(proofService *services.ProofService, p2pNode *p2p.Node, nodeService *services.NodeService, interval time.Duration, difficulty int) *ProofScheduler {
    return &ProofScheduler{
        proofService: proofService,
        p2pNode:      p2pNode,
        nodeService:  nodeService,
        interval:     interval,
        difficulty:   difficulty,
        stopChan:     make(chan bool),
    }
}

// Start begins the proof scheduling
func (s *ProofScheduler) Start() {
    log.Printf("Starting proof scheduler (interval: %v, difficulty: %d)", s.interval, s.difficulty)
    
    ticker := time.NewTicker(s.interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                s.run()
            case <-s.stopChan:
                ticker.Stop()
                return
            }
        }
    }()
}

// Stop halts the proof scheduler
func (s *ProofScheduler) Stop() {
    close(s.stopChan)
}

func (s *ProofScheduler) run() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    log.Println("Running proof challenge cycle...")
    
    // Get all active chunks with their assignments
    chunks, err := s.getActiveChunks(ctx)
    if err != nil {
        log.Printf("Failed to get active chunks: %v", err)
        return
    }
    
    if len(chunks) == 0 {
        return
    }
    
    log.Printf("Scheduling proof challenges for %d chunks", len(chunks))
    
    // Stagger challenges to avoid overwhelming nodes
    for i, chunk := range chunks {
        go func(c ChunkAssignment, delay time.Duration) {
            time.Sleep(delay)
            s.sendChallenge(ctx, c)
        }(chunk, time.Duration(i)*100*time.Millisecond)
    }
}

type ChunkAssignment struct {
    ChunkID uuid.UUID
    NodeID  uuid.UUID
    PeerID  string
}

func (s *ProofScheduler) getActiveChunks(ctx context.Context) ([]ChunkAssignment, error) {
    // Get all active chunk assignments
    rows, err := s.db.Pool.Query(ctx, `
        SELECT c.id, ca.node_id, sn.peer_id
        FROM chunks c
        JOIN chunk_assignments ca ON c.id = ca.chunk_id
        JOIN storage_nodes sn ON ca.node_id = sn.id
        WHERE ca.status = 'active' 
        AND sn.status = 'active'
        AND (c.last_verified_at IS NULL OR c.last_verified_at < $1)
    `, time.Now().Add(-4*time.Hour)) // Challenge each chunk every 4 hours
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var assignments []ChunkAssignment
    for rows.Next() {
        var a ChunkAssignment
        err := rows.Scan(&a.ChunkID, &a.NodeID, &a.PeerID)
        if err != nil {
            continue
        }
        assignments = append(assignments, a)
    }
    
    return assignments, nil
}

func (s *ProofScheduler) sendChallenge(ctx context.Context, assignment ChunkAssignment) {
    // Create challenge in database
    challenge, err := s.proofService.CreateChallenge(ctx, assignment.ChunkID, assignment.NodeID)
    if err != nil {
        log.Printf("Failed to create challenge for chunk %s: %v", assignment.ChunkID, err)
        return
    }
    
    // Send challenge to node
    req := &pb.ProofChallengeRequest{
        ChallengeId: challenge.ID.String(),
        ChunkId:     assignment.ChunkID.String(),
        Seed:        challenge.Seed,
        Difficulty:  int32(challenge.Difficulty),
    }
    
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    start := time.Now()
    resp, err := s.p2pNode.SendProofChallenge(ctx, assignment.PeerID, req)
    duration := time.Since(start)
    
    if err != nil {
        log.Printf("Proof challenge failed for chunk %s on node %s: %v", 
            assignment.ChunkID, assignment.PeerID, err)
        
        // Mark as failed due to timeout
        s.proofService.MarkChallengeFailed(ctx, challenge.ID, "timeout")
        return
    }
    
    // Verify proof
    err = s.proofService.VerifyProof(ctx, challenge.ID, resp.ProofHash, int(duration.Milliseconds()))
    if err != nil {
        log.Printf("Proof verification failed for chunk %s on node %s: %v",
            assignment.ChunkID, assignment.PeerID, err)
    } else {
        log.Printf("Proof verified for chunk %s on node %s (duration: %dms)",
            assignment.ChunkID, assignment.PeerID, duration.Milliseconds())
        
        // Update chunk last_verified_at
        s.db.Pool.Exec(ctx,
            "UPDATE chunks SET last_verified_at = $1 WHERE id = $2",
            time.Now(), assignment.ChunkID)
    }
}
```

### 4.2 Enhanced Proof Service

**File**: `coordinator/internal/services/proof.go`

Add verification improvements:

```go
// VerifyProof verifies a proof response from a storage node with full validation
func (s *ProofService) VerifyProof(ctx context.Context, challengeID uuid.UUID, proofHash string, durationMs int) error {
    // Get challenge
    var challenge models.ProofChallenge
    err := s.db.Pool.QueryRow(ctx,
        "SELECT id, chunk_id, node_id, seed, difficulty FROM proof_challenges WHERE id = $1",
        challengeID).Scan(&challenge.ID, &challenge.ChunkID, &challenge.NodeID, &challenge.Seed, &challenge.Difficulty)
    if err != nil {
        return fmt.Errorf("challenge not found: %w", err)
    }
    
    // Verify timing (should complete within 2 seconds)
    if durationMs > 2000 {
        s.MarkChallengeFailed(ctx, challengeID, "timeout")
        return fmt.Errorf("proof verification timed out (%dms)", durationMs)
    }
    
    // Get chunk data for verification
    chunk, err := s.getChunkForVerification(ctx, challenge.ChunkID)
    if err != nil {
        return fmt.Errorf("failed to get chunk data: %w", err)
    }
    
    // Generate expected proof (using actual chunk data)
    expectedHash := s.generateProofWithData(challenge.Seed, chunk.Data, challenge.Difficulty)
    
    if proofHash != expectedHash {
        s.MarkChallengeFailed(ctx, challengeID, "invalid_proof")
        return fmt.Errorf("invalid proof hash: expected %s, got %s", expectedHash, proofHash)
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

// MarkChallengeFailed marks a challenge as failed
func (s *ProofService) MarkChallengeFailed(ctx context.Context, challengeID uuid.UUID, reason string) error {
    _, err := s.db.Pool.Exec(ctx,
        "UPDATE proof_challenges SET status = 'failed', error = $1, verified_at = $2 WHERE id = $3",
        reason, time.Now(), challengeID)
    return err
}

// generateProofWithData generates proof using actual chunk data
func (s *ProofService) generateProofWithData(seed []byte, chunkData []byte, difficulty int) string {
    // Combine seed with chunk data
    data := append(seed, chunkData...)
    
    // Perform sequential hashing
    for i := 0; i < difficulty; i++ {
        hash := sha256.Sum256(data)
        data = hash[:]
    }
    
    return hex.EncodeToString(data)
}

func (s *ProofService) getChunkForVerification(ctx context.Context, chunkID uuid.UUID) (*models.Chunk, error) {
    var chunk models.Chunk
    var data []byte
    err := s.db.Pool.QueryRow(ctx,
        "SELECT id, file_id, hash, size_bytes, data FROM chunks WHERE id = $1",
        chunkID).Scan(&chunk.ID, &chunk.FileID, &chunk.Hash, &chunk.SizeBytes, &data)
    if err != nil {
        return nil, err
    }
    chunk.Data = data
    return &chunk, nil
}

// GetNodeProofStats retrieves proof statistics for a node
func (s *ProofService) GetNodeProofStats(ctx context.Context, nodeID uuid.UUID, since time.Time) (*ProofStats, error) {
    var stats ProofStats
    err := s.db.Pool.QueryRow(ctx,
        `SELECT 
            COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified,
            COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
            COUNT(*) as total,
            COALESCE(AVG(duration_ms), 0) as avg_duration
         FROM proof_challenges 
         WHERE node_id = $1 AND created_at >= $2`,
        nodeID, since).Scan(&stats.Verified, &stats.Failed, &stats.Total, &stats.AvgDurationMs)
    
    if stats.Total > 0 {
        stats.SuccessRate = float64(stats.Verified) / float64(stats.Total) * 100
    } else {
        stats.SuccessRate = 100
    }
    
    return &stats, err
}

type ProofStats struct {
    Verified      int
    Failed        int
    Total         int
    SuccessRate   float64
    AvgDurationMs float64
}
```

### 4.3 Update P2P Node

**File**: `coordinator/internal/p2p/node.go`

Add proof challenge method:

```go
// SendProofChallenge sends a proof challenge to a storage node
func (n *Node) SendProofChallenge(ctx context.Context, peerID string, req *pb.ProofChallengeRequest) (*pb.ProofChallengeResponse, error) {
    pid, err := peer.Decode(peerID)
    if err != nil {
        return nil, fmt.Errorf("invalid peer ID: %w", err)
    }
    
    stream, err := n.host.NewStream(ctx, pid, "/federated-storage/1.0.0/proof-challenge")
    if err != nil {
        return nil, fmt.Errorf("failed to open stream: %w", err)
    }
    defer stream.Close()
    
    // Write request
    if err := writeProofChallengeRequest(stream, req); err != nil {
        return nil, err
    }
    
    // Read response
    resp, err := readProofChallengeResponse(stream)
    if err != nil {
        return nil, err
    }
    
    return resp, nil
}
```

---

## Phase 5: Node Economics (Week 6)

### 5.1 Database Migration

**New file**: `coordinator/migrations/4_economics.up.sql`

```sql
-- Add uptime tracking fields
ALTER TABLE storage_nodes ADD COLUMN IF NOT EXISTS expected_heartbeats_24h INTEGER DEFAULT 0;
ALTER TABLE storage_nodes ADD COLUMN IF NOT EXISTS actual_heartbeats_24h INTEGER DEFAULT 0;
ALTER TABLE storage_nodes ADD COLUMN IF NOT EXISTS last_earnings_calculation DATE;

-- Create earnings calculation indices
CREATE INDEX IF NOT EXISTS idx_node_earnings_date ON node_earnings(date);
CREATE INDEX IF NOT EXISTS idx_storage_nodes_earnings ON storage_nodes(last_earnings_calculation);
```

### 5.2 Economics Service

**New file**: `coordinator/internal/services/economics.go`

```go
package services

import (
    "context"
    "fmt"
    "math"
    "time"
    
    "github.com/federated-storage/coordinator/internal/storage"
    "github.com/google/uuid"
)

// EconomicsService handles node earnings calculations
type EconomicsService struct {
    db            *storage.DB
    storageCredit int64 // credits per GB per month
    dailyRate     float64
}

// EarningsResult represents calculated earnings for a node
type EarningsResult struct {
    NodeID             uuid.UUID
    Date               time.Time
    StorageBytes       int64
    StorageCredits     int64
    UptimePercentage   float64
    UptimePenalty      int64
    ProofSuccessRate   float64
    ProofPenalty       int64
    TotalEarnings      int64
}

// NewEconomicsService creates a new economics service
func NewEconomicsService(db *storage.DB, storageCreditPerGBMonth int64) *EconomicsService {
    // Convert monthly rate to daily rate
    dailyRate := float64(storageCreditPerGBMonth) / 30.0
    
    return &EconomicsService{
        db:            db,
        storageCredit: storageCreditPerGBMonth,
        dailyRate:     dailyRate,
    }
}

// CalculateDailyEarnings calculates earnings for a node for a specific date
func (s *EconomicsService) CalculateDailyEarnings(ctx context.Context, nodeID uuid.UUID, date time.Time) (*EarningsResult, error) {
    result := &EarningsResult{
        NodeID: nodeID,
        Date:   date,
    }
    
    // Get storage used
    storageBytes, err := s.getNodeStorageBytes(ctx, nodeID, date)
    if err != nil {
        return nil, fmt.Errorf("failed to get storage: %w", err)
    }
    result.StorageBytes = storageBytes
    
    // Calculate base storage credits
    gb := float64(storageBytes) / (1024 * 1024 * 1024)
    result.StorageCredits = int64(gb * s.dailyRate)
    
    // Calculate uptime percentage
    uptime, err := s.calculateUptime(ctx, nodeID, date)
    if err != nil {
        return nil, fmt.Errorf("failed to calculate uptime: %w", err)
    }
    result.UptimePercentage = uptime
    
    // Apply uptime penalty if < 95%
    if uptime < 95.0 {
        penaltyRate := (95.0 - uptime) / 100.0
        result.UptimePenalty = int64(float64(result.StorageCredits) * penaltyRate)
    }
    
    // Get proof success rate
    proofStats, err := s.getProofStats(ctx, nodeID, date)
    if err != nil {
        return nil, fmt.Errorf("failed to get proof stats: %w", err)
    }
    result.ProofSuccessRate = proofStats.SuccessRate
    
    // Apply proof penalty if < 95%
    if proofStats.SuccessRate < 95.0 {
        penaltyRate := (95.0 - proofStats.SuccessRate) / 100.0
        result.ProofPenalty = int64(float64(result.StorageCredits) * penaltyRate)
    }
    
    // Calculate total
    result.TotalEarnings = result.StorageCredits - result.UptimePenalty - result.ProofPenalty
    if result.TotalEarnings < 0 {
        result.TotalEarnings = 0
    }
    
    return result, nil
}

// RecordEarnings saves earnings to database and updates node balance
func (s *EconomicsService) RecordEarnings(ctx context.Context, earnings *EarningsResult) error {
    tx, err := s.db.Pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Insert earnings record
    _, err = tx.Exec(ctx,
        `INSERT INTO node_earnings 
         (node_id, date, storage_bytes, storage_credits, uptime_penalty, missed_proof_penalty, total_earnings)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         ON CONFLICT (node_id, date) DO UPDATE SET
            storage_bytes = EXCLUDED.storage_bytes,
            storage_credits = EXCLUDED.storage_credits,
            uptime_penalty = EXCLUDED.uptime_penalty,
            missed_proof_penalty = EXCLUDED.missed_proof_penalty,
            total_earnings = EXCLUDED.total_earnings`,
        earnings.NodeID, earnings.Date, earnings.StorageBytes, earnings.StorageCredits,
        earnings.UptimePenalty, earnings.ProofPenalty, earnings.TotalEarnings)
    if err != nil {
        return fmt.Errorf("failed to record earnings: %w", err)
    }
    
    // Update node earned_credits
    _, err = tx.Exec(ctx,
        "UPDATE storage_nodes SET earned_credits = earned_credits + $1, last_earnings_calculation = $2 WHERE id = $3",
        earnings.TotalEarnings, earnings.Date, earnings.NodeID)
    if err != nil {
        return fmt.Errorf("failed to update node credits: %w", err)
    }
    
    return tx.Commit(ctx)
}

// CalculateUptimePercentage calculates uptime from heartbeats
func (s *EconomicsService) CalculateUptimePercentage(ctx context.Context, nodeID uuid.UUID) (float64, error) {
    // Get expected heartbeats (one every 30 seconds)
    expectedHeartbeats := int64(24 * 60 * 60 / 30) // 2880 per day
    
    // Count actual heartbeats in last 24 hours
    var actualHeartbeats int64
    err := s.db.Pool.QueryRow(ctx,
        `SELECT COUNT(*) FROM node_heartbeats 
         WHERE node_id = $1 AND created_at >= $2`,
        nodeID, time.Now().Add(-24*time.Hour)).Scan(&actualHeartbeats)
    if err != nil {
        return 0, err
    }
    
    uptime := (float64(actualHeartbeats) / float64(expectedHeartbeats)) * 100
    if uptime > 100 {
        uptime = 100
    }
    
    return uptime, nil
}

func (s *EconomicsService) getNodeStorageBytes(ctx context.Context, nodeID uuid.UUID, date time.Time) (int64, error) {
    var totalBytes int64
    err := s.db.Pool.QueryRow(ctx,
        `SELECT COALESCE(SUM(c.size_bytes), 0)
         FROM chunk_assignments ca
         JOIN chunks c ON ca.chunk_id = c.id
         WHERE ca.node_id = $1 
         AND ca.status = 'active'
         AND ca.created_at <= $2`,
        nodeID, date.Add(24*time.Hour)).Scan(&totalBytes)
    return totalBytes, err
}

func (s *EconomicsService) calculateUptime(ctx context.Context, nodeID uuid.UUID, date time.Time) (float64, error) {
    startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
    endOfDay := startOfDay.Add(24 * time.Hour)
    
    // Expected heartbeats: one every 30 seconds
    expected := int64(24 * 60 * 60 / 30)
    
    // Count actual heartbeats for that day
    var actual int64
    err := s.db.Pool.QueryRow(ctx,
        `SELECT COUNT(*) FROM node_heartbeats 
         WHERE node_id = $1 AND created_at >= $2 AND created_at < $3`,
        nodeID, startOfDay, endOfDay).Scan(&actual)
    if err != nil {
        return 0, err
    }
    
    uptime := (float64(actual) / float64(expected)) * 100
    if uptime > 100 {
        uptime = 100
    }
    
    return uptime, nil
}

func (s *EconomicsService) getProofStats(ctx context.Context, nodeID uuid.UUID, date time.Time) (*ProofStats, error) {
    startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
    endOfDay := startOfDay.Add(24 * time.Hour)
    
    var verified, total int64
    err := s.db.Pool.QueryRow(ctx,
        `SELECT 
            COUNT(CASE WHEN status = 'verified' THEN 1 END),
            COUNT(*)
         FROM proof_challenges 
         WHERE node_id = $1 AND created_at >= $2 AND created_at < $3`,
        nodeID, startOfDay, endOfDay).Scan(&verified, &total)
    if err != nil {
        return nil, err
    }
    
    successRate := 100.0
    if total > 0 {
        successRate = (float64(verified) / float64(total)) * 100
    }
    
    return &ProofStats{SuccessRate: successRate}, nil
}

type ProofStats struct {
    SuccessRate float64
}
```

### 5.3 Daily Earnings Job

**New file**: `coordinator/internal/jobs/economics.go`

```go
package jobs

import (
    "context"
    "log"
    "time"
    
    "github.com/federated-storage/coordinator/internal/services"
)

// DailyEarningsJob calculates earnings for all nodes daily
type DailyEarningsJob struct {
    economicsService *services.EconomicsService
    nodeService      *services.NodeService
    stopChan         chan bool
}

// NewDailyEarningsJob creates a new daily earnings job
func NewDailyEarningsJob(economicsService *services.EconomicsService, nodeService *services.NodeService) *DailyEarningsJob {
    return &DailyEarningsJob{
        economicsService: economicsService,
        nodeService:      nodeService,
        stopChan:         make(chan bool),
    }
}

// Start begins the daily earnings calculation
func (j *DailyEarningsJob) Start() {
    log.Println("Starting daily earnings job")
    
    // Calculate when next midnight UTC is
    now := time.Now().UTC()
    nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
    durationUntilMidnight := nextMidnight.Sub(now)
    
    // Wait until midnight for first run
    go func() {
        select {
        case <-time.After(durationUntilMidnight):
            j.run()
            
            // Then run every 24 hours
            ticker := time.NewTicker(24 * time.Hour)
            for {
                select {
                case <-ticker.C:
                    j.run()
                case <-j.stopChan:
                    ticker.Stop()
                    return
                }
            }
        case <-j.stopChan:
            return
        }
    }()
}

// Stop halts the earnings job
func (j *DailyEarningsJob) Stop() {
    close(j.stopChan)
}

func (j *DailyEarningsJob) run() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    yesterday := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour)
    log.Printf("Calculating daily earnings for %s", yesterday.Format("2006-01-02"))
    
    // Get all active nodes
    nodes, err := j.nodeService.GetAllNodes(ctx)
    if err != nil {
        log.Printf("Failed to get nodes: %v", err)
        return
    }
    
    for _, node := range nodes {
        // Skip if already calculated for this date
        if node.LastEarningsCalculation != nil && node.LastEarningsCalculation.Equal(yesterday) {
            continue
        }
        
        earnings, err := j.economicsService.CalculateDailyEarnings(ctx, node.ID, yesterday)
        if err != nil {
            log.Printf("Failed to calculate earnings for node %s: %v", node.ID, err)
            continue
        }
        
        err = j.economicsService.RecordEarnings(ctx, earnings)
        if err != nil {
            log.Printf("Failed to record earnings for node %s: %v", node.ID, err)
            continue
        }
        
        log.Printf("Node %s earnings: %d credits (storage: %d, uptime_penalty: %d, proof_penalty: %d)",
            node.ID, earnings.TotalEarnings, earnings.StorageCredits, 
            earnings.UptimePenalty, earnings.ProofPenalty)
    }
    
    log.Printf("Daily earnings calculation complete for %d nodes", len(nodes))
}
```

### 5.4 Heartbeat Tracking Table

**Update migration**: `coordinator/migrations/4_economics.up.sql`

```sql
-- Heartbeat tracking for uptime calculation
CREATE TABLE IF NOT EXISTS node_heartbeats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES storage_nodes(id) ON DELETE CASCADE,
    used_storage_bytes BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_heartbeats_node_id ON node_heartbeats(node_id);
CREATE INDEX IF NOT EXISTS idx_node_heartbeats_created_at ON node_heartbeats(created_at);
```

---

## Phase 6: File Deletion Propagation (Week 7)

### 6.1 Enhanced Delete Handler

**File**: `coordinator/internal/handlers/file.go`

```go
func (h *FileHandler) DeleteFile(c *gin.Context) {
    // ... existing validation code ...
    
    // Get all chunk assignments before deleting
    chunks, err := h.chunkService.GetChunksByFile(c.Request.Context(), fileID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get file chunks"})
        return
    }
    
    // Mark file as deleting (soft delete)
    err = h.fileService.MarkFileDeleting(c.Request.Context(), fileID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // Send delete commands to nodes asynchronously
    go func() {
        ctx := context.Background()
        
        for _, chunk := range chunks {
            assignments, err := h.chunkService.GetChunkAssignments(ctx, chunk.ID)
            if err != nil {
                continue
            }
            
            for _, assignment := range assignments {
                // Get node peer ID
                node, err := h.nodeService.GetNode(ctx, assignment.NodeID)
                if err != nil {
                    continue
                }
                
                // Send delete command
                ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
                err = h.p2pNode.SendDeleteCommand(ctx, node.PeerID, chunk.ID.String())
                cancel()
                
                if err == nil {
                    // Mark assignment as deleted
                    h.chunkService.UpdateAssignmentStatus(ctx, chunk.ID, node.ID, "deleted")
                }
            }
            
            // Mark chunk as pending cleanup
            h.chunkService.MarkChunkPendingCleanup(ctx, chunk.ID)
        }
    }()
    
    c.JSON(http.StatusOK, gin.H{
        "status": "deleting",
        "message": "File marked for deletion. Cleanup in progress.",
    })
}
```

### 6.2 Delete Command Handler (Storage Node)

**File**: `storage-node/internal/p2p/node.go`

Add delete handler:

```go
// SetChunkDeleteHandler sets up the handler for deleting chunks
func (n *Node) SetChunkDeleteHandler(handler func(*pb.DeleteChunkRequest) (*pb.DeleteChunkResponse, error)) {
    n.host.SetStreamHandler("/federated-storage/1.0.0/delete-chunk", func(s network.Stream) {
        defer s.Close()
        
        req, err := readDeleteChunkRequest(s)
        if err != nil {
            sendErrorResponse(s, err)
            return
        }
        
        resp, err := handler(req)
        if err != nil {
            sendErrorResponse(s, err)
            return
        }
        
        writeDeleteChunkResponse(s, resp)
    })
}
```

**File**: `storage-node/cmd/storage-node/main.go`

```go
p2pNode.SetChunkDeleteHandler(func(req *pb.DeleteChunkRequest) (*pb.DeleteChunkResponse, error) {
    err := chunkService.DeleteChunk(req.ChunkId)
    if err != nil {
        return &pb.DeleteChunkResponse{Success: false}, err
    }
    return &pb.DeleteChunkResponse{Success: true}, nil
})
```

### 6.3 Cleanup Job

**New file**: `coordinator/internal/jobs/cleanup.go`

```go
package jobs

import (
    "context"
    "log"
    "time"
    
    "github.com/federated-storage/coordinator/internal/services"
)

// CleanupJob handles periodic cleanup of deleted files and orphaned chunks
type CleanupJob struct {
    fileService  *services.FileService
    chunkService *services.ChunkService
    interval     time.Duration
    stopChan     chan bool
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(fileService *services.FileService, chunkService *services.ChunkService, interval time.Duration) *CleanupJob {
    return &CleanupJob{
        fileService:  fileService,
        chunkService: chunkService,
        interval:     interval,
        stopChan:     make(chan bool),
    }
}

// Start begins the cleanup job
func (j *CleanupJob) Start() {
    log.Printf("Starting cleanup job (interval: %v)", j.interval)
    
    ticker := time.NewTicker(j.interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                j.run()
            case <-j.stopChan:
                ticker.Stop()
                return
            }
        }
    }()
}

// Stop halts the cleanup job
func (j *CleanupJob) Stop() {
    close(j.stopChan)
}

func (j *CleanupJob) run() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    log.Println("Running cleanup job...")
    
    // 1. Clean up files marked for deletion > 24h ago
    j.cleanupDeletedFiles(ctx)
    
    // 2. Clean up orphaned chunks
    j.cleanupOrphanedChunks(ctx)
    
    log.Println("Cleanup job complete")
}

func (j *CleanupJob) cleanupDeletedFiles(ctx context.Context) {
    // Find files marked 'deleting' for > 24 hours
    files, err := j.fileService.GetFilesPendingCleanup(ctx, 24*time.Hour)
    if err != nil {
        log.Printf("Failed to get files pending cleanup: %v", err)
        return
    }
    
    for _, file := range files {
        // Check if all chunks have been deleted or marked orphaned
        canDelete, err := j.fileService.CanSafelyDelete(ctx, file.ID)
        if err != nil {
            continue
        }
        
        if canDelete {
            err := j.fileService.PermanentlyDelete(ctx, file.ID)
            if err != nil {
                log.Printf("Failed to delete file %s: %v", file.ID, err)
            } else {
                log.Printf("Permanently deleted file %s", file.ID)
            }
        }
    }
}

func (j *CleanupJob) cleanupOrphanedChunks(ctx context.Context) {
    // Find chunks with no active assignments
    chunks, err := j.chunkService.GetOrphanedChunks(ctx)
    if err != nil {
        log.Printf("Failed to get orphaned chunks: %v", err)
        return
    }
    
    for _, chunk := range chunks {
        // Delete chunk data from coordinator's database
        err := j.chunkService.DeleteChunkData(ctx, chunk.ID)
        if err != nil {
            log.Printf("Failed to delete chunk %s data: %v", chunk.ID, err)
        }
    }
    
    log.Printf("Cleaned up %d orphaned chunks", len(chunks))
}
```

---

## Phase 7: Integration & Testing (Week 8)

### 7.1 Integration Tests

**Update file**: `tests/integration/api_test.go`

Add comprehensive tests:

```go
// TestChunkDistribution tests end-to-end chunk distribution
func TestChunkDistribution(t *testing.T) {
    // Setup: Start coordinator + 3 storage nodes
    // 1. Upload file
    // 2. Verify chunks distributed to all 3 nodes
    // 3. Verify each node has the chunk data on disk
}

// TestChunkRetrieval tests downloading from storage nodes
func TestChunkRetrieval(t *testing.T) {
    // Setup: Upload file distributed to nodes
    // 1. Download file
    // 2. Verify chunks retrieved from storage nodes, not coordinator
    // 3. Verify data integrity
}

// TestReReplication tests automatic replication when node fails
func TestReReplication(t *testing.T) {
    // Setup: Upload file with 3 replicas
    // 1. Kill one storage node
    // 2. Wait for replication job
    // 3. Verify chunk is replicated to a new node
}

// TestProofChallenges tests proof-of-storage system
func TestProofChallenges(t *testing.T) {
    // Setup: Upload and distribute chunks
    // 1. Trigger proof challenge
    // 2. Verify node responds correctly
    // 3. Verify proof is validated
    // 4. Check earnings calculation
}

// TestFileDeletion tests file deletion propagation
func TestFileDeletion(t *testing.T) {
    // Setup: Upload file distributed to nodes
    // 1. Delete file
    // 2. Verify delete commands sent to nodes
    // 3. Verify chunks removed from node storage
    // 4. Verify cleanup after 24h
}

// TestConcurrentUploads tests multiple simultaneous uploads
func TestConcurrentUploads(t *testing.T) {
    // Upload 10 files concurrently
    // Verify all succeed
    // Verify proper chunk distribution
    // Check no data corruption
}
```

### 7.2 Main Function Updates

**File**: `coordinator/cmd/api/main.go`

Add job initialization:

```go
func main() {
    // ... existing setup code ...
    
    // Initialize replication service
    replicationService := services.NewReplicationService(db, p2pNode, chunkService, nodeService)
    
    // Start background jobs
    replicationJob := jobs.NewReplicationJob(replicationService, cfg.Storage.DefaultReplicas, 5*time.Minute)
    replicationJob.Start()
    defer replicationJob.Stop()
    
    proofScheduler := jobs.NewProofScheduler(proofService, p2pNode, nodeService, 4*time.Hour, cfg.Storage.ProofDifficulty)
    proofScheduler.Start()
    defer proofScheduler.Stop()
    
    economicsService := services.NewEconomicsService(db, cfg.Storage.StorageCreditPerGBMonth)
    dailyEarningsJob := jobs.NewDailyEarningsJob(economicsService, nodeService)
    dailyEarningsJob.Start()
    defer dailyEarningsJob.Stop()
    
    cleanupJob := jobs.NewCleanupJob(fileService, chunkService, 1*time.Hour)
    cleanupJob.Start()
    defer cleanupJob.Stop()
    
    // ... rest of main function ...
}
```

---

## Migration Summary

### Database Migrations Required

1. **Migration 3**: Add replication tracking, assignment statuses, relay addresses
2. **Migration 4**: Add node_heartbeats table, earnings indices

### Configuration Changes

Add to `config.toml`:

```toml
[p2p]
enable_relay = true
relay_addresses = [
    "/ip4/relay1.example.com/tcp/4001/p2p/12D3KooW...",
    "/ip4/relay2.example.com/tcp/4001/p2p/12D3KooW..."
]
is_relay_client = false  # Set to true if behind NAT

[jobs]
replication_interval = "5m"
proof_challenge_interval = "4h"
cleanup_interval = "1h"
```

---

## Testing Checklist

### Before Deployment

- [ ] All unit tests pass
- [ ] Integration tests pass with 3+ nodes
- [ ] Chunk distribution works end-to-end
- [ ] Downloads retrieve from storage nodes
- [ ] Re-replication triggers when node fails
- [ ] Proof challenges execute and validate
- [ ] Earnings calculated correctly
- [ ] File deletion propagates to all nodes
- [ ] NAT traversal works with relay
- [ ] Graceful shutdown of all components
- [ ] Error handling works for all failure modes

### Performance Targets

- [ ] Upload speed: 100MB file in < 30 seconds
- [ ] Download speed: 100MB file in < 30 seconds
- [ ] Proof response time: < 2 seconds
- [ ] Replication lag: < 10 minutes for failed node
- [ ] Cleanup lag: < 24 hours for deleted files
- [ ] Concurrent uploads: Support 10+ simultaneous

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| P2P transfer failures | Retry with exponential backoff, fall back to coordinator |
| Node churn | Rapid re-replication, maintain coordinator backups |
| Network partition | Uptime tracking, graceful degradation |
| Data corruption | Hash validation at every step |
| Memory leaks | Context timeouts, connection pooling |
| Race conditions | Database transactions, atomic operations |

---

## Success Criteria

By end of implementation:

- [ ] Chunks distributed to 3+ storage nodes via P2P
- [ ] Downloads retrieve from storage nodes, not just coordinator
- [ ] Automatic re-replication when nodes fail
- [ ] Proof challenges every 4 hours per chunk
- [ ] Daily earnings calculation based on storage/uptime/proofs
- [ ] File deletion propagated to all storage nodes
- [ ] NAT traversal with relay nodes
- [ ] All integration tests passing
- [ ] 10+ nodes can participate simultaneously

---

## Appendix: Protocol Buffer Code Generation

To generate Go code from protobuf definitions:

```bash
# Install protoc-gen-go
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       coordinator/proto/storage.proto
```

---

*Document Version: 1.0*  
*Created: 2026-02-19*  
*Status: Ready for Implementation*
