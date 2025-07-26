package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ChatProtocolID = "/chat/1.0.0"

type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

type ChatManager struct {
	username string
	host     host.Host
	active   sync.Map // peer.ID -> bool (to avoid duplicate write loops)
}

func NewChatManager(h host.Host, username string) *ChatManager {
	cm := &ChatManager{
		host:     h,
		username: username,
	}
	h.SetStreamHandler(ChatProtocolID, cm.handleIncomingStream)
	return cm
}

// This is called when another peer initiates a chat stream
func (cm *ChatManager) handleIncomingStream(s network.Stream) {
	remote := s.Conn().RemotePeer()
	fmt.Printf("\n📥 Incoming chat stream from [%s]\n", remote.ShortString())
	go cm.readLoop(s)
}

// This is called after match is successful, and you want to chat
func (cm *ChatManager) StartChat(ctx context.Context, remote peer.ID) {
	// Prevent duplicate write loops
	if _, exists := cm.active.LoadOrStore(remote, true); exists {
		fmt.Printf("⚠️ Already chatting with [%s]\n", remote.ShortString())
		return
	}

	// Make sure we're connected
	err := cm.host.Connect(ctx, peer.AddrInfo{ID: remote})
	if err != nil {
		fmt.Println("❌ Connection failed:", err)
		return
	}

	// Outgoing stream
	s, err := cm.host.NewStream(ctx, remote, ChatProtocolID)
	if err != nil {
		fmt.Println("❌ Failed to create outgoing stream:", err)
		return
	}

	fmt.Printf("📤 Chat started with [%s]\n", remote.ShortString())

	go cm.writeLoop(s)
	// Note: readLoop will be triggered from handler when remote initiates
}

func (cm *ChatManager) readLoop(s network.Stream) {
	remote := s.Conn().RemotePeer()
	dec := json.NewDecoder(s)
	for {
		var msg Message
		err := dec.Decode(&msg)
		if err != nil {
			fmt.Printf("📴 [%s] left the chat or error: %v\n", remote.ShortString(), err)
			return
		}
		fmt.Printf("\n👤 %s: %s\nYou: ", msg.Username, msg.Text)
	}
}

func (cm *ChatManager) writeLoop(s network.Stream) {
	enc := json.NewEncoder(s)
	stdin := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		text, err := stdin.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Input error:", err)
			return
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		msg := Message{
			Username: cm.username,
			Text:     text,
		}

		err = enc.Encode(msg)
		if err != nil {
			fmt.Println("❌ Failed to send message:", err)
			return
		}
	}
}
