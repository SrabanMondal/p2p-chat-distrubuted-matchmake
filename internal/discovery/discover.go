package discovery

import (
	"context"
	"fmt"
	"time"
	"os"
	"os/signal"
	"syscall"
	"github.com/SrabanMondal/p2pchat/internal/utils"
	kademlia "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)
func Discover(host host.Host, name string, token string, dht *kademlia.IpfsDHT, ctx context.Context){
	var mm MatchManager
	mm = *NewMatchManager(host,name)
	host.SetStreamHandler(MatchProtoID, mm.HandleMatch)

	cid := utils.RendezvousCID(token)

	fmt.Println("📢 Providing ",token," token on DHT...")
	if err := dht.Provide(ctx, cid, true); err != nil {
		panic(err)
	}
	fmt.Println("✅ Token successfully advertised to DHT.")
	// defer cancel()

	found := false
	for info := range dht.FindProvidersAsync(ctx, cid, 0) {
		if info.ID == host.ID() {
			found = true
			fmt.Printf("🔁 Found provider (YOU): %s\n", info.ID)
			fmt.Println("🎯 Verified: You are discoverable on DHT.")
			break
		}
	}
	if !found {
		fmt.Println("⚠️ Provided token not yet discoverable. Wait or retry.")
	}
	ctx2, _ := context.WithTimeout(ctx, time.Second*30)
	fmt.Println("🔍 Searching for token on DHT...")
	var triedPeers = make(map[peer.ID]bool)
	go func() {
		for {
			if mm.GetState() == StatePaired {
				fmt.Println("Pairing done")
				break
			}
			for info := range dht.FindProvidersAsync(ctx2, cid, 0) {
				if info.ID == host.ID()  {
					fmt.Println("bad peer: ",info.ID.ShortString())
					continue
				}
				fmt.Println("✅ Found new peer to try:", info.ID)
				if mm.GetState() == StatePaired {
					fmt.Println("Pairing done")
					break
				}
				triedPeers[info.ID] = true
				go mm.TryToPairWith(ctx2, info)
			}
			//fmt.Println("Scanning going on...")
			time.Sleep(1 * time.Second)
		}
		defer fmt.Println("Scanning stopped")
	}()
	defer fmt.Println("Discover done")
	fmt.Println("libp2p host is running. Press Ctrl+C to exit.")

	// Keep the application running until a termination signal is received
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigCh

	fmt.Println("Shutting down libp2p host...")
}