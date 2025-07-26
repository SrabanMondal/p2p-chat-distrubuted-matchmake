package utils

import (
	"fmt"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)


func LoadOrCreateKey() (crypto.PrivKey, error) {
	// Load from disk or generate if not exists
	if data, err := os.ReadFile("peerkey"); err == nil {
		privKey, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, err
		}
		return privKey, nil
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}

	data, _ := crypto.MarshalPrivateKey(priv)
	os.WriteFile("peerkey", data, 0600)

	return priv, nil
}

func RendezvousCID(name string) cid.Cid {
    h, err := mh.Sum([]byte(name), mh.SHA2_256, -1)
    if err != nil {
        panic(err)
    }
    return cid.NewCidV1(cid.Raw, h)
}

func ConvertToAddrInfo(addrs []ma.Multiaddr) []peer.AddrInfo {
    var infos []peer.AddrInfo
    for _, addr := range addrs {
        info, err := peer.AddrInfoFromP2pAddr(addr)
        if err != nil {
            fmt.Printf("Error converting multiaddr: %v", err)
            continue
        }
        infos = append(infos, *info)
    }
    return infos
}