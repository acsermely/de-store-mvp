# Federated Storage Network - MVP Implementation Plan

**Timeline:** 2-3 months (12 weeks)  
**Team:** Solo developer  
**Approach:** Full horizontal slice - get end-to-end functionality working early, then iterate

---

## Executive Summary

This plan implements both the **Coordinator** and **Storage Node** components in parallel, with a focus on getting a complete end-to-end flow working as early as possible. Each phase delivers a working slice of functionality.

---

## Phase 1: Foundation (Weeks 1-2)

**Goal:** Set up project structure, database schemas, and basic connectivity

### Week 1: Project Setup & Database

#### Coordinator (Go)
```
coordinator/
├── cmd/api/
│   └── main.go                    # Basic HTTP server setup
├── internal/
│   ├── config/
│   │   └── config.go              # TOML config loading
│   └── storage/
│       └── postgres.go            # Database connection & migrations
├── migrations/
│   ├── 001_initial_schema.sql     # Users, files, nodes, chunks tables
│   └── 002_credit_system.sql      # Credit purchases table
├── go.mod
└── docker-compose.yml             # PostgreSQL for development
```

**Tasks:**
- [ ] Initialize Go module: `go mod init github.com/yourorg/coordinator`
- [ ] Set up PostgreSQL connection with pgx/pgxpool
- [ ] Create migration files for all tables from spec
- [ ] Implement config loading from TOML
- [ ] Create Docker Compose for local development

**Dependencies:** None

#### Storage Node (Go)
```
storage-node/
├── cmd/storage-node/
│   └── main.go                    # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go              # TOML config loading
│   └── storage/
│       └── sqlite.go              # SQLite setup
├── migrations/
│   └── 001_initial_schema.sql     # Config and chunks tables
├── go.mod
└── config.example.toml
```

**Tasks:**
- [ ] Initialize Go module: `go mod init github.com/yourorg/storage-node`
- [ ] Set up SQLite with schema
- [ ] Implement config loading from TOML
- [ ] Create CLI structure with `init`, `start`, `chunks list`, `drain` commands

**Dependencies:** None

### Week 2: Basic Connectivity

#### Coordinator
**Tasks:**
- [ ] Set up libp2p host with TCP + QUIC transports
- [ ] Implement DHT (Kademlia) for peer discovery
- [ ] Create basic HTTP server with middleware (logging, CORS)
- [ ] Implement health check endpoint: `GET /health`

#### Storage Node
**Tasks:**
- [ ] Set up libp2p host with TCP + QUIC transports
- [ ] Implement bootstrap connection to coordinator
- [ ] Create admin API on localhost: `GET /health`, `GET /chunks`
- [ ] Implement node registration flow (generate keys, register with coordinator)

**Integration Checkpoint:**
- Storage node can register with coordinator
- Both can discover each other via DHT
- Basic heartbeat working

---

## Phase 2: Core Storage Flow (Weeks 3-5)

**Goal:** Implement file upload, chunking, and distribution

### Week 3: Upload Initiation & Chunking

#### Coordinator
**Tasks:**
- [ ] Implement JWT authentication middleware
- [ ] Create auth endpoints: `POST /api/v1/auth/register`, `POST /api/v1/auth/login`
- [ ] Implement password hashing with bcrypt
- [ ] Create upload initiation endpoint: `POST /api/v1/files/upload/initiate`
  - Generate encryption key
  - Split file into 256KB chunks
  - Assign chunk IDs
  - Return upload session ID
- [ ] Implement chunk metadata storage

**Key Implementation:**
```go
// internal/services/upload.go
func (s *UploadService) InitiateUpload(userID uuid.UUID, filename string, sizeBytes int64) (*UploadSession, error) {
    // Generate encryption key
    key := make([]byte, 32)
    rand.Read(key)
    
    // Calculate chunks
    chunkCount := int(math.Ceil(float64(sizeBytes) / ChunkSizeBytes))
    
    // Create upload session
    session := &UploadSession{
        ID: uuid.New(),
        UserID: userID,
        Chunks: make([]ChunkInfo, chunkCount),
    }
    
    // Return encrypted key to client
    return session, nil
}
```

#### Storage Node
**Tasks:**
- [ ] Implement chunk storage filesystem layout (two-level directory)
- [ ] Create chunk store methods: `Store()`, `Retrieve()`, `Delete()`
- [ ] Implement hash validation on store
- [ ] Set up admin endpoint: `GET /chunks` to list stored chunks

### Week 4: Chunk Upload & Distribution

#### Coordinator
**Tasks:**
- [ ] Implement chunk upload endpoint: `POST /api/v1/files/upload/:id/chunk`
  - Accept chunk data
  - Encrypt with AES-256-GCM
  - Select nodes for storage (3 replicas minimum)
  - Send chunks to storage nodes via libp2p
- [ ] Create node selection algorithm (simple round-robin for MVP)
- [ ] Implement chunk distribution via libp2p to storage nodes
- [ ] Track chunk assignments in database

**Protocol Buffers:**
```protobuf
// coordinator/proto/storage.proto
message StoreChunkRequest {
    string chunk_id = 1;
    string file_id = 2;
    int32 chunk_index = 3;
    string hash = 4;
    int32 size_bytes = 5;
    bytes data = 6;
}
```

#### Storage Node
**Tasks:**
- [ ] Implement libp2p protocol handler: `/federated-storage/1.0.0/store-chunk`
- [ ] Accept chunks from coordinator
- [ ] Validate and store chunks
- [ ] Update SQLite metadata
- [ ] Return acknowledgment

**Integration Checkpoint:**
- Client can initiate upload
- Coordinator can receive and encrypt chunks
- Chunks are distributed to 3+ storage nodes
- Storage nodes persist chunks to disk

### Week 5: Upload Completion & Download

#### Coordinator
**Tasks:**
- [ ] Implement upload completion: `POST /api/v1/files/upload/:id/complete`
  - Verify all chunks received
  - Deduct credits from user
  - Mark file as ready
- [ ] Implement credit system: credit balance check, deduction
- [ ] Create download endpoint: `GET /api/v1/files/:id/download`
  - Retrieve chunk locations from database
  - Fetch chunks from storage nodes via libp2p
  - Decrypt and reassemble file
  - Stream to client
- [ ] Implement file listing: `GET /api/v1/files`

#### Storage Node
**Tasks:**
- [ ] Implement libp2p protocol handler: `/federated-storage/1.0.0/retrieve-chunk`
- [ ] Read chunks from disk
- [ ] Return chunk data to coordinator
- [ ] Update heartbeat to report chunk count and storage usage

**Integration Checkpoint:**
- Complete upload flow: initiate → upload chunks → complete
- Complete download flow: request → fetch chunks → decrypt → deliver
- Credit system working (deduction on upload)

---

## Phase 3: Proof-of-Storage System (Weeks 6-7)

**Goal:** Implement proof challenges to verify nodes are actually storing data

### Week 6: Proof Generation

#### Storage Node
**Tasks:**
- [ ] Implement proof engine with sequential hashing
- [ ] Create proof generation method:
  ```go
  func (e *ProofEngine) GenerateProof(chunkID string, seed []byte, difficulty int) (*ProofResult, error)
  ```
- [ ] Implement libp2p handler: `/federated-storage/1.0.0/proof-challenge`
- [ ] Return proof hash and duration (no validation on node side)

**Protocol Buffers:**
```protobuf
message ProofChallenge {
    string challenge_id = 1;
    string chunk_id = 2;
    bytes seed = 3;
    int32 difficulty = 4;
}

message ProofResponse {
    string challenge_id = 1;
    string proof_hash = 2;
    int64 duration_ms = 3;
}
```

#### Coordinator
**Tasks:**
- [ ] Create proof challenge scheduler (every 4 hours per chunk)
- [ ] Implement proof verification logic
- [ ] Validate response time (< 2 seconds)
- [ ] Validate proof hash correctness
- [ ] Track proof results in database

### Week 7: Node Economics & Penalties

#### Coordinator
**Tasks:**
- [ ] Implement node economics service
- [ ] Calculate daily earnings based on:
  - Storage credits (1 credit/GB/month)
  - Uptime penalties (< 95% uptime)
  - Missed proof penalties
  - Failed challenge penalties
- [ ] Create earnings calculation job (runs daily)
- [ ] Implement balance endpoint: `GET /api/v1/balance`
- [ ] Update node reputation scores based on performance

**Key Implementation:**
```go
func (s *NodeEconomicsService) CalculateDailyEarnings(
    ctx context.Context,
    nodeID uuid.UUID,
) (*EarningsResult, error) {
    // Calculate storage credits
    // Apply penalties for downtime, missed proofs
    // Update node.earned_credits
}
```

**Integration Checkpoint:**
- Coordinator sends proof challenges to storage nodes
- Storage nodes generate proofs and return timing
- Coordinator validates proofs and updates earnings
- Penalty system working

---

## Phase 4: Credit System & Node Management (Weeks 8-9)

**Goal:** Complete credit system, node management, and reliability features

### Week 8: Credit System

#### Coordinator
**Tasks:**
- [ ] Implement credit purchase endpoint (mock): `POST /api/v1/auth/credits/purchase`
- [ ] Add credits to user balance
- [ ] Create credit transaction history
- [ ] Implement storage cost calculation:
  ```go
  func CalculateStorageCost(sizeBytes int64, replicaCount int) int64
  ```
- [ ] Display credits in user responses

#### Storage Node
**Tasks:**
- [ ] Display earned credits in heartbeat
- [ ] Add `drain` command to admin API
- [ ] Implement graceful shutdown (finish active operations)

### Week 9: Re-replication & Reliability

#### Coordinator
**Tasks:**
- [ ] Implement node health monitoring (via heartbeats)
- [ ] Create re-replication job:
  - Detect missing chunks (node down)
  - Select new nodes for replicas
  - Copy chunks from healthy nodes
  - Update chunk assignments
- [ ] Implement file deletion with chunk cleanup
- [ ] Send delete commands to storage nodes via libp2p

#### Storage Node
**Tasks:**
- [ ] Implement libp2p handler: `/federated-storage/1.0.0/delete-chunk`
- [ ] Delete chunks from filesystem and database
- [ ] Handle drain mode (reject new chunks, finish active)

**Integration Checkpoint:**
- Credit purchase and storage cost calculation working
- Node drain mode functional
- Re-replication when nodes fail
- File deletion propagates to all nodes

---

## Phase 5: Polish & Deployment (Weeks 10-12)

**Goal:** Testing, documentation, and production readiness

### Week 10: Testing & Bug Fixes

**Tasks:**
- [ ] Write integration tests for upload/download flow
- [ ] Test proof challenge system
- [ ] Test re-replication scenarios
- [ ] Test credit system edge cases
- [ ] Load test with multiple concurrent uploads
- [ ] Test NAT traversal (run nodes in different networks)
- [ ] Fix bugs and edge cases

### Week 11: Documentation & CLI Polish

#### Coordinator
**Tasks:**
- [ ] Write API documentation (OpenAPI/Swagger)
- [ ] Document environment variables
- [ ] Create deployment guide
- [ ] Add structured logging

#### Storage Node
**Tasks:**
- [ ] Improve CLI output and help text
- [ ] Add progress indicators for long operations
- [ ] Document configuration options
- [ ] Create setup guide for node operators

### Week 12: Deployment & Final Integration

**Tasks:**
- [ ] Create production Dockerfiles
- [ ] Set up CI/CD pipeline
- [ ] Deploy coordinator to staging
- [ ] Deploy test storage nodes
- [ ] End-to-end testing in staging environment
- [ ] Performance benchmarking
- [ ] Final security review

---

## Technical Dependencies & Ordering

### Must Complete Before Next Phase:

**Phase 1 → Phase 2:**
- Database schemas must be complete
- libp2p connectivity must work
- Basic HTTP server must be running

**Phase 2 → Phase 3:**
- File upload/download must work end-to-end
- Chunk distribution to multiple nodes must work
- Heartbeat must be reporting

**Phase 3 → Phase 4:**
- Proof challenges must work
- Timing validation must be accurate
- Earnings calculation must be correct

**Phase 4 → Phase 5:**
- Credit system must be functional
- Node management must work
- Re-replication must handle failures

---

## Daily/Weekly Rhythm

### Recommended Schedule:

**Monday:**
- Review last week's progress
- Adjust plan if needed
- Start new features

**Tuesday-Thursday:**
- Deep work on implementation
- Daily commits
- Test as you go

**Friday:**
- Integration testing
- Documentation updates
- Plan next week

### Solo Developer Tips:

1. **Work on both components in the same week** - Don't isolate too long or integration will be painful
2. **Use Docker Compose** for local testing with multiple storage nodes
3. **Commit working states frequently** - Tag milestones
4. **Test integration early and often** - Don't wait until the end
5. **Keep a simple test script** that uploads and downloads a file

---

## Integration Checkpoints

After each phase, verify these end-to-end flows:

### Phase 1 Checkpoint:
```bash
# Coordinator running
curl http://localhost:8080/health

# Storage node can register
./storage-node init --name "Test Node" --coordinator-url http://localhost:8080
./storage-node start

# Coordinator sees the node
curl http://localhost:8080/api/v1/nodes  # (add this endpoint for testing)
```

### Phase 2 Checkpoint:
```bash
# Register user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -d '{"email":"test@example.com","password":"test123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"test@example.com","password":"test123"}'

# Upload a file (multipart form)
curl -X POST http://localhost:8080/api/v1/files/upload/initiate \
  -H "Authorization: Bearer <token>" \
  -d '{"filename":"test.txt","size_bytes":1024}'

# Download the file
curl http://localhost:8080/api/v1/files/<file_id>/download \
  -H "Authorization: Bearer <token>"
```

### Phase 3 Checkpoint:
```bash
# Check node received proof challenges
./storage-node chunks list

# Check coordinator proof logs
# Verify earnings calculated
```

### Phase 4 Checkpoint:
```bash
# Purchase credits
curl -X POST http://localhost:8080/api/v1/auth/credits/purchase \
  -H "Authorization: Bearer <token>" \
  -d '{"amount_usd":10}'

# Check node balance
curl http://localhost:8080/api/v1/balance \
  -H "X-API-Key: <node_api_key>"

# Test re-replication: kill a node, verify chunks moved
```

---

## Key Metrics to Track

- **Upload speed:** Time to upload 100MB file
- **Download speed:** Time to download 100MB file
- **Proof response time:** Should be < 2 seconds
- **Chunk distribution:** All chunks have 3+ replicas
- **Node uptime tracking:** Accurate within 1%
- **Credit calculations:** Accurate to 2 decimal places

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| libp2p complexity | Start with basic connectivity, add features incrementally |
| Database migrations | Use migration tool (golang-migrate), test upgrades |
| NAT traversal issues | Test early with nodes on different networks |
| Performance issues | Benchmark at Phase 2, optimize in Phase 5 |
| Scope creep | Stick to MVP features, create "v1.1" backlog |

---

## Success Criteria

By end of Week 12, you should have:

- [ ] Working coordinator deployed
- [ ] 3+ storage nodes running and connected
- [ ] End-to-end file upload and download
- [ ] Proof-of-storage challenges working
- [ ] Credit system functional
- [ ] Automatic re-replication on node failure
- [ ] Documentation for operators
- [ ] Basic monitoring and logging

---

## Post-MVP Backlog

Move these to v1.1:
- Real payment processing (Stripe)
- Web UI for file management
- Node operator dashboard
- Advanced erasure coding
- Bandwidth optimization
- Mobile client
- Federation (multiple coordinators)

---

*Plan Version: 1.0*  
*Created: 2025-02-18*  
*Status: Ready for Implementation*
