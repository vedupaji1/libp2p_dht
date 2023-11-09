package main

import (
	"context"

	logger "github.com/inconshreveable/log15"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectionManager struct{}

var _ connmgr.ConnManager = (*ConnectionManager)(nil)

func (ConnectionManager) TagPeer(peer.ID, string, int)             {}
func (ConnectionManager) UntagPeer(peer.ID, string)                {}
func (ConnectionManager) UpsertTag(peer.ID, string, func(int) int) {}
func (ConnectionManager) GetTagInfo(peer.ID) *connmgr.TagInfo      { return &connmgr.TagInfo{} }
func (ConnectionManager) TrimOpenConns(ctx context.Context)        {}
func (ConnectionManager) Notifee() network.Notifiee                { return &Notifiee{} }
func (ConnectionManager) Protect(peer.ID, string)                  {}
func (ConnectionManager) Unprotect(peer.ID, string) bool           { return false }
func (ConnectionManager) IsProtected(peer.ID, string) bool         { return false }
func (ConnectionManager) Close() error                             { return nil }

type Notifiee struct{}

var _ network.Notifiee = (*Notifiee)(nil)

func (nn *Notifiee) Connected(n network.Network, c network.Conn) {
	logger.Info("Connected To Node", "NodeId", c.RemotePeer(), "LatestNodeList", n.Peers())
	stream, err := c.NewStream(context.Background())
	if err != nil {
		logger.Error("Failed To Create A Stream", "Error", err)
	} else {
		stream.Write([]byte("Hello Bro"))
	}
}
func (nn *Notifiee) Disconnected(n network.Network, c network.Conn) {
	logger.Info("Disconnected From Node", "NodeId", c.RemotePeer(), "LatestNodeList", n.Peers())
}
func (nn *Notifiee) Listen(n network.Network, addr ma.Multiaddr) {
	logger.Info("Listening To Node", "Addr", addr)
}
func (nn *Notifiee) ListenClose(n network.Network, addr ma.Multiaddr) {
	logger.Info("Listening Closed To Node", "Addr", addr)
}
