package main

import (
	"context"
	"fmt"
	"net/http"

	logger "github.com/inconshreveable/log15"
	libp2pDHT "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

type NodeService struct {
	host *host.Host
	dht  *libp2pDHT.IpfsDHT
}

type PublishDataArgs struct {
	Key  string
	Data string
}
type PublishDataReply struct{}

type RetrieveDataArgs struct {
	Key  string
	Data string
}
type RetrieveDataReply struct {
	Data string
}

func (node *NodeService) PublishData(r *http.Request, req *PublishDataArgs, res *PublishDataReply) error {
	logger.Info("Received `PublishData` Request", "RemoteAddr", r.RemoteAddr)
	err := node.dht.PutValue(context.Background(), fmt.Sprintf("/%s/%s", ValidatorNameSpace, req.Key), []byte(req.Data))
	if err != nil {
		logger.Error("Failed To Put Value", "Error", err)
	}
	logger.Info("Data Stored", "Key", req.Key, "Data", req.Data)
	return nil
}

func (node *NodeService) RetrieveData(r *http.Request, req *RetrieveDataArgs, res *RetrieveDataReply) error {
	logger.Info("Received `RetrieveData` Request", "RemoteAddr", r.RemoteAddr)
	data, err := node.dht.GetValue(context.Background(), fmt.Sprintf("/%s/%s", ValidatorNameSpace, req.Key))
	if err != nil {
		logger.Error("Failed To Get Data", "Error", err)
	}
	res.Data = string(data)
	logger.Info("Data Found", "Data", string(data))
	return nil
}
