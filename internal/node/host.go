package node

import (
	"context"
	"fmt"
	"time"
	"github.com/SrabanMondal/p2pchat/internal/utils"
	"github.com/libp2p/go-libp2p"
	kademlia "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)
const RelayRendezvous = "/libp2p/relay"


func CreateHost(ctx context.Context, t,q int) (host.Host, *kademlia.IpfsDHT, error) {
	var dht *kademlia.IpfsDHT
	priv, _ := utils.LoadOrCreateKey()
	node, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", t),
    fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", q),
		),

		libp2p.NATPortMap(),       
		libp2p.EnableNATService(),

		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.DefaultMuxers,

		libp2p.Ping(true),

		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dht, err = kademlia.New(ctx, h,
				kademlia.Mode(kademlia.ModeAuto),
				kademlia.BootstrapPeers(utils.ConvertToAddrInfo(kademlia.DefaultBootstrapPeers)...),
			)
			return dht, err
		}),

		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithPeerSource(
			func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				if dht == nil {
					panic("DHT not initialized for autorelay peer source!")
				}
				return dht.FindProvidersAsync(ctx, utils.RendezvousCID("/libp2p/relay"), numPeers)
			},
					autorelay.WithMinCandidates(4),          // Aim for at least 3 potential relay candidates
					autorelay.WithMaxCandidates(10),         // Don't search for too many, balance efficiency
					autorelay.WithNumRelays(2),              // Try to maintain active connections to 2 relays
					autorelay.WithBootDelay(5*time.Second), // Give the DHT some time to bootstrap before aggressive relay search
				),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Bootstrap the DHT
	// This initiates the process of connecting to bootstrap peers and populating the routing table.
	fmt.Println("Starting Kademlia DHT bootstrap...")
	if err = dht.Bootstrap(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}
	fmt.Println("Kademlia DHT bootstrapped successfully.")

	fmt.Println("--- Host Information ---")
	fmt.Printf("🟢 Host ID: %s\n", node.ID())
	fmt.Println("🟢 Host addresses (listening):")
	for _, addr := range node.Addrs() {
		fmt.Printf("  - %s\n", addr)
	}
	
	fmt.Println("\n-------------------------------------")
	for _, pi := range utils.ConvertToAddrInfo(kademlia.DefaultBootstrapPeers) {
		fmt.Printf("🔗 Bootstrap Peer: %s - %v\n", pi.ID.ShortString(), pi.Addrs)
	}
	fmt.Println("-------------------------------------")
	fmt.Println("Searching for bootstrap peers to establish connection. Wait 10 seconds")
	time.Sleep(10 * time.Second)
	fmt.Println("-----------------------------")
	return node, dht, nil
}