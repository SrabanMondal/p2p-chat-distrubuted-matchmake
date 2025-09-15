package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"github.com/SrabanMondal/p2pchat/internal/chat"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerState int32

const MatchProtoID = "/match/1.0.0/chat-public-111"

const (
	StateIdle PeerState = iota
	StateRequesting
	StateRequestReceived
	StateAccepted
	StatePaired
)

type MatchManager struct {
	state     PeerState
	stateMu   sync.Mutex
	matchedTo peer.ID
	host      host.Host
	chat *chat.ChatManager
}

func NewMatchManager(h host.Host,username string) *MatchManager {
	return &MatchManager{
		state: StateIdle,
		host:  h,
		chat:  chat.NewChatManager(h, username),
	}
}

func (mm *MatchManager) HandleMatch(s network.Stream) {
	defer s.Close()
	remotePeer := s.Conn().RemotePeer()

	buf := make([]byte, 128)
	n, err := s.Read(buf)
	if err != nil {
		fmt.Println("❌ Failed to read MATCH:", err)
		return
	}
	message := strings.TrimSpace(string(buf[:n]))
	fmt.Println("📩 Received from", remotePeer.ShortString(), ":", message)

	if message != "MATCH" {
		fmt.Println("❌ Invalid message in HandleMatch:", message)
		return
	}

	mm.stateMu.Lock()
	if mm.state != StateIdle {
		fmt.Println("❌ Busy. Rejecting MATCH from", remotePeer.ShortString())
		s.Write([]byte("REJECT\n"))
		mm.stateMu.Unlock()
		return
	}
	mm.state = StateRequestReceived
	mm.matchedTo = remotePeer
	mm.stateMu.Unlock()

	// Step 1: Send ACCEPT
	s.Write([]byte("ACCEPT\n"))

	// Step 2: Wait for ACCEPTED
	buf2 := make([]byte, 128)
	s.SetReadDeadline(time.Now().Add(5 * time.Second))
	n2, err := s.Read(buf2)
	if err != nil {
		fmt.Println("❌ No ACCEPTED received:", err)
		mm.resetToIdle()
		return
	}
	if strings.TrimSpace(string(buf2[:n2])) != "ACCEPTED" {
		fmt.Println("❌ Invalid handshake from", remotePeer.ShortString())
		mm.resetToIdle()
		return
	}

	// Step 3: Final confirmation
	s.Write([]byte("ACCEPTED\n"))
	mm.stateMu.Lock()
	mm.state = StatePaired
	mm.stateMu.Unlock()
	fmt.Println("🤝 Handshake complete. Matched with", remotePeer.ShortString())
	go mm.startChatPlaceholder(remotePeer)
}

func (mm *MatchManager) TryToPairWith(ctx context.Context, p peer.AddrInfo) {
	match_s, err := mm.host.NewStream(ctx, p.ID, MatchProtoID)
	if err != nil {
		fmt.Println("❌ Cannot open stream to", p.ID.ShortString(), ":", err)
		return
	}
	defer match_s.Close()

	mm.stateMu.Lock()
	if mm.state != StateIdle {
		mm.stateMu.Unlock()
		return
	}
	mm.state = StateRequesting
	mm.matchedTo = p.ID
	mm.stateMu.Unlock()

	fmt.Println("📤 Sending MATCH to", p.ID.ShortString())
	match_s.Write([]byte("MATCH\n"))

	// Step 1: Wait for ACCEPT
	buf := make([]byte, 128)
	match_s.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := match_s.Read(buf)
	if err != nil {
		fmt.Println("❌ No ACCEPT from", p.ID.ShortString(), ":", err)
		mm.resetToIdle()
		return
	}
	if strings.TrimSpace(string(buf[:n])) != "ACCEPT" {
		fmt.Println("❌ Unexpected response from", p.ID.ShortString(), ":", string(buf[:n]))
		mm.resetToIdle()
		return
	}

	// Step 2: Send ACCEPTED
	match_s.Write([]byte("ACCEPTED\n"))

	// Step 3: Wait for final ACCEPTED
	buf2 := make([]byte, 128)
	match_s.SetReadDeadline(time.Now().Add(5 * time.Second))
	n2, err := match_s.Read(buf2)
	if err != nil || strings.TrimSpace(string(buf2[:n2])) != "ACCEPTED" {
		fmt.Println("❌ Final confirmation failed from", p.ID.ShortString())
		mm.resetToIdle()
		return
	}

	fmt.Println("🤝 Match confirmed with", p.ID.ShortString())
	mm.stateMu.Lock()
	mm.state = StateAccepted
	mm.stateMu.Unlock()
	mm.state = StatePaired
	go mm.startChatPlaceholder(p.ID)
}

func (mm *MatchManager) startChatPlaceholder(remotePeer peer.ID) {
	fmt.Println("💬 [Placeholder] Starting chat with", remotePeer.ShortString())
	ctx := context.Background()
	mm.chat.StartChat(ctx, remotePeer)
}

func (mm *MatchManager) resetToIdle() {
	mm.stateMu.Lock()
	defer mm.stateMu.Unlock()
	mm.state = StateIdle
	mm.matchedTo = ""
}

func (mm *MatchManager) GetState() PeerState {
	mm.stateMu.Lock()
	defer mm.stateMu.Unlock()
	return mm.state
}
