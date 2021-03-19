package discovery

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/reddec/tinc-boot/tincd/daemon"
)

const discoveryPort = "18655"

func New(ssd *SSD, config *daemon.Config, interval time.Duration) *Discovery {
	return &Discovery{
		client:        NewClient(ssd, config, interval),
		serverHandler: NewServer(ssd, config),
	}
}

type Discovery struct {
	client        *Client
	serverHandler http.Handler
	httpServer    struct {
		server *http.Server
		done   chan struct{}
	}
}

func (ds *Discovery) Configured(payload daemon.Configuration) {
	ds.httpServer.server = &http.Server{
		Addr:    fmt.Sprint(payload.IP, ":", discoveryPort),
		Handler: ds.serverHandler,
	}
	ds.httpServer.done = make(chan struct{})

	go func() {
		defer close(ds.httpServer.done)
		log.Println("discovery service started on", ds.httpServer.server.Addr)
		err := ds.httpServer.server.ListenAndServe()
		if err != nil {
			log.Println("discovery server stopped:", err)
		}
	}()
}

func (ds *Discovery) Stopped(payload daemon.Configuration) {
	if ds.httpServer.server != nil {
		_ = ds.httpServer.server.Close()
		<-ds.httpServer.done
	}
}

func (ds *Discovery) SubnetAdded(payload daemon.EventSubnetAdded) {
	if ds.client.Watch(context.Background(), strings.Split(payload.Peer.Subnet, "/")[0]+":"+discoveryPort) {
		log.Println("watching subnet", payload.Peer.Subnet)
	}
}

func (ds *Discovery) SubnetRemoved(payload daemon.EventSubnetRemoved) {
	log.Println("forgetting subnet", payload.Peer.Subnet)
	ds.client.Forget(strings.Split(payload.Peer.Subnet, "/")[0] + ":" + discoveryPort)
}

func (ds *Discovery) Ready(payload daemon.EventReady) {

}
