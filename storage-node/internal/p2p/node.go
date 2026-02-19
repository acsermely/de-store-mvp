package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Node represents a libp2p storage node
type Node struct {
	host   host.Host
	dht    *dht.IpfsDHT
	config NodeConfig
}

// NodeConfig holds P2P node configuration
type NodeConfig struct {
	ListenAddresses []string
	BootstrapPeers  []string
}

// NewNode creates a new libp2p node
func NewNode(listenAddresses []string) (*Node, error) {
	if len(listenAddresses) == 0 {
		listenAddresses = []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		}
	}

	config := NodeConfig{
		ListenAddresses: listenAddresses,
	}

	return &Node{
		config: config,
	}, nil
}

// Start starts the P2P node
func (n *Node) Start() error {
	// Build libp2p options
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(n.config.ListenAddresses...),
	}

	// Create host
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %w", err)
	}
	n.host = h

	// Create DHT for peer discovery
	ctx := context.Background()
	kadDHT, err := dht.New(ctx, h)
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}
	n.dht = kadDHT

	// Bootstrap DHT
	if err := kadDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	return nil
}

// Stop stops the P2P node
func (n *Node) Stop() error {
	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			return err
		}
	}
	if n.host != nil {
		return n.host.Close()
	}
	return nil
}

// Close is an alias for Stop
func (n *Node) Close() error {
	return n.Stop()
}

// Host returns the libp2p host
func (n *Node) Host() host.Host {
	return n.host
}

// ID returns the peer ID
func (n *Node) ID() peer.ID {
	if n.host == nil {
		return ""
	}
	return n.host.ID()
}

// IDString returns the peer ID as a string
func (n *Node) IDString() string {
	return n.ID().String()
}

// Addrs returns the multiaddrs the node is listening on
func (n *Node) Addrs() []string {
	if n.host == nil {
		return nil
	}

	var addrs []string
	for _, addr := range n.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), n.ID().String()))
	}
	return addrs
}

// Connect connects to a peer
func (n *Node) Connect(ctx context.Context, peerAddr string) error {
	addrInfo, err := peer.AddrInfoFromString(peerAddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer address: %w", err)
	}

	if err := n.host.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// SetStreamHandler sets a handler for a protocol
func (n *Node) SetStreamHandler(protocolID string, handler network.StreamHandler) {
	n.host.SetStreamHandler(protocol.ID(protocolID), handler)
}

// SetChunkStoreHandler sets up the handler for storing chunks
func (n *Node) SetChunkStoreHandler(handler func(chunkID string, data []byte) error) {
	n.host.SetStreamHandler("/federated-storage/1.0.0/store-chunk", func(s network.Stream) {
		defer s.Close()
		// In a full implementation, read chunk ID and data from stream
		// For MVP, simplified
	})
}

// SetChunkRetrieveHandler sets up the handler for retrieving chunks
func (n *Node) SetChunkRetrieveHandler(handler func(chunkID string) ([]byte, error)) {
	n.host.SetStreamHandler("/federated-storage/1.0.0/retrieve-chunk", func(s network.Stream) {
		defer s.Close()
		// In a full implementation, read chunk ID and return data
		// For MVP, simplified
	})
}

// SetProofChallengeHandler sets up the handler for proof challenges
func (n *Node) SetProofChallengeHandler(handler func(chunkID string, seed []byte, difficulty int) (string, int64, error)) {
	n.host.SetStreamHandler("/federated-storage/1.0.0/proof-challenge", func(s network.Stream) {
		defer s.Close()
		// In a full implementation, read challenge and return proof
		// For MVP, simplified
	})
}
