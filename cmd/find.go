package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SrabanMondal/p2pchat/internal/discovery"
	"github.com/SrabanMondal/p2pchat/internal/node"
	"github.com/SrabanMondal/p2pchat/internal/utils"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/spf13/cobra"
)

var (
	token string
	name string
)
var findCmd = &cobra.Command{
	Use: "find",
	Short: "Find a random stranger to talk. Establishes a 1 on 1 communication",
	Long: "Finds either a random stranger or provide token flag to find person with same token.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		t,q := 10000, 20000
		host, dht, err := node.CreateHost(ctx,t,q)
		defer dht.Close()
		defer host.Close()
		utils.CheckError(err)
	    fmt.Println("listening on: ",host.Addrs())
		fmt.Println("DHT established: ",dht.PeerID())
		fmt.Println("Checking connected peers...")
		for _, p := range host.Network().Peers() {
			fmt.Printf("✅ Connected to: %s\n", p.ShortString())
		}
		for _, c := range host.Network().Conns() {
			if c.Stat().Direction == network.DirInbound {
				continue
			}
			fmt.Printf("🔁 Connected to peer %s over transport %s\n", c.RemotePeer(), c.ConnState().Transport)
		}
		discovery.Discover(host, name, token, dht, ctx)
		fmt.Println("P2P chat is running. Press Ctrl+C to exit.")

		// Keep the application running until a termination signal is received
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sigCh

		fmt.Println("Shutting down P2P chat...")
		},
}

func init(){
	findCmd.Flags().StringVarP(&token,"token","t","chat-public-p2pchat","Provide token to connect")
	findCmd.Flags().StringVarP(&name,"name","n","user","Provide an alias to use as username for this chat session")
	rootCmd.AddCommand(findCmd)
}