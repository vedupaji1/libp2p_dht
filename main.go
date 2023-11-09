package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	gorillaRPC "github.com/gorilla/rpc"
	gorillaJSON "github.com/gorilla/rpc/json"
	logger "github.com/inconshreveable/log15"
	"github.com/libp2p/go-libp2p"
	libp2pDHT "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	discoveryRouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	discoveryUtil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cast"

	// "github.com/libp2p/go-libp2p/core/peerstore"

	viperPKG "github.com/spf13/viper"
	// manet "github.com/multiformats/go-multiaddr/net"
)

const (
	Rendezvous           = "TempOp"
	ProtocolPrefixForDHT = "TempOpDHT"
	ValidatorNameSpace   = "temp"
)

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

type NodeConfigData struct {
	Port               int
	NodeKey            *crypto.PrivKey
	BootstrappingPeers []interface{}
}

func getNodeConfig(configPath string) *NodeConfigData {
	viper := viperPKG.New()
	viper.SetConfigFile(configPath)
	viper.ReadInConfig()
	port := viper.GetInt("port")
	nodeKeyStr := viper.GetString("nodeKey")
	var nodeKey crypto.PrivKey
	if nodeKeyStr == "" {
		randNodeKey, _, err := crypto.GenerateKeyPair(
			crypto.Ed25519,
			-1,
		)
		if err != nil {
			log.Panic("Failed To Generate Node Credentials: ", err)
		}
		nodeKey = randNodeKey
		nodeKeyMarshaled, err := crypto.MarshalPrivateKey(nodeKey)
		if err != nil {
			logger.Error("Failed To Marshal NodeKey", "Error", err)
			log.Panic("Failed To Marshal NodeKey", err)
		}
		logger.Info("Random Node Key Is Generated", "nodeKey", hex.EncodeToString(nodeKeyMarshaled))
	} else {
		nodeKeyMarshaled, err := hex.DecodeString(nodeKeyStr)
		if err != nil {
			logger.Error("Failed To UnMarshal NodeKey", "Error", err)
			log.Panic("Failed To UnMarshal NodeKey", err)
		}
		nodeKey, err = crypto.UnmarshalPrivateKey(nodeKeyMarshaled)
		if err != nil {
			logger.Error("Failed To Recover NodeKey", "Error", err)
			log.Panic("Failed To Recover NodeKey", err)
		}
	}
	nodePubKeyMarshaled, err := crypto.MarshalPublicKey(nodeKey.GetPublic())
	if err != nil {
		logger.Error("Failed To Marshal NodePubKey", "Error", err)
		log.Panic("Failed To Marshal NodeKey", err)
	}
	logger.Info("", "NodePubKey", hex.EncodeToString(nodePubKeyMarshaled))
	return &NodeConfigData{
		Port:               port,
		NodeKey:            &nodeKey,
		BootstrappingPeers: cast.ToSlice(viper.Get("bootstrappingPeers")),
	}
}

func startRPCServer(port int, node *host.Host, dht *libp2pDHT.IpfsDHT) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("Node Is Running"))
		res.WriteHeader(http.StatusOK)
	})
	rpcServer := gorillaRPC.NewServer()
	nodeService := new(NodeService)
	nodeService.host = node
	nodeService.dht = dht
	rpcServer.RegisterCodec(gorillaJSON.NewCodec(), "application/json")
	rpcServer.RegisterService(nodeService, "")
	router.Handle("/node", rpcServer)
	rpcPort := fmt.Sprintf("%v", port+1)
	logger.Info("Node RPC Server Is Running", "Port", rpcPort)
	http.ListenAndServe(":"+rpcPort, router)
}

func main() {
	configPath := flag.String("configPath", "", "Node Config File Path")
	flag.Parse()
	if *configPath == "" {
		log.Panic("Config File Path Is Not Specified")
	}
	nodeConfigData := getNodeConfig(*configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node, err := libp2p.New(
		libp2p.Identity(*nodeConfigData.NodeKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", nodeConfigData.Port),
		),
		libp2p.ConnectionManager(ConnectionManager{}),
		libp2p.DisableRelay())

	if err != nil {
		log.Panic("Failed To Create Node: ", err)
	}
	logger.Info("Node Is Running", "NodeId", node.ID(), "NodeAddr", node.Addrs())

	dht, err := libp2pDHT.New(context.Background(), node, libp2pDHT.ProtocolPrefix(ProtocolPrefixForDHT), libp2pDHT.Mode(libp2pDHT.ModeServer), libp2pDHT.NamespacedValidator(ValidatorNameSpace, blankValidator{}))
	if err != nil {
		logger.Error("Failed To Create DHT", "Error", err)
		log.Panic("Failed To Create DHT:", err)
	}
	if err = dht.Bootstrap(ctx); err != nil {
		logger.Error("Failed to Bootstrap DHT", "Error", err)
		log.Panic("Failed To Bootstrap:", err)
	}

	for _, value := range nodeConfigData.BootstrappingPeers {
		fmt.Println("Bootstrapping Peers", nodeConfigData.BootstrappingPeers)
		peerData := cast.ToStringMapString(value)
		nodeId, err := peer.Decode(peerData["nodeid"])
		if err != nil {
			logger.Error("Failed To Get NodeId Ins", "Error", err)
			log.Panic("Failed To Get NodeId Ins: ", err)
		}
		nodeAddr, err := ma.NewMultiaddr(peerData["nodeaddr"])
		if err != nil {
			logger.Error("Failed To Get MultiAddr Ins", "Error", err)
			log.Panic("Failed To Get MultiAddr Ins: ", err)
		}
		node.Peerstore().AddAddr(nodeId, nodeAddr, peerstore.PermanentAddrTTL)
		if err != nil {
			logger.Error("Failed To Get AddrInfo Ins", "Error", err)
			log.Panic("Failed To Get AddrInfo Ins: ", err)
		}
		err = node.Connect(context.Background(), node.Peerstore().PeerInfo(nodeId))
		if err != nil {
			logger.Error("Failed To Connect To Node", "NodeId", peerData["nodeid"], "Error", err)
			log.Panic("Failed To Connect To Node:", err)
		}
	}

	var routingDiscovery = discoveryRouting.NewRoutingDiscovery(dht)
	discoveryUtil.Advertise(ctx, routingDiscovery, Rendezvous)
	peers, err := routingDiscovery.FindPeers(ctx, Rendezvous)
	if err != nil {
		log.Fatal(err)
	}
	for p := range peers {
		fmt.Println("PeerInfo: ", p)
		if p.ID == node.ID() {
			continue
		}
		if node.Network().Connectedness(p.ID) != network.Connected {
			err = node.Connect(ctx, p)
			if err != nil {
				continue
			}
		}
	}
	time.Sleep(time.Second * time.Duration(1))
	startRPCServer(nodeConfigData.Port, &node, dht)
	<-ctx.Done()

}
