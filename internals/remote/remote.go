package remote

import (
	"context"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type Connection struct {
	peer        *webrtc.PeerConnection
	httpServer  *http.Server
	DataChannel *webrtc.DataChannel
	handlers    map[string]Handler

	outBuffer []*Response
	stopped   bool
	stopChan  chan any
}

var originAllowList = []string{
	"http://localhost:3000",
	"https://minepkg.io",
	"https://preview.minepkg.io",
	"https://dev.minepkg.io",
}

func isOriginAllowed(origin string) bool {
	for _, allowed := range originAllowList {
		if origin == allowed {
			return true
		}
	}
	return false
}

func New() *Connection {
	return &Connection{
		peer:     nil,
		handlers: make(map[string]Handler),
		stopChan: make(chan any),
	}
}

func (c *Connection) ListenAndServe() {
	for !c.stopped {
		log.Println("## handshake request")
		c.ListenForHandshake(context.Background())
		if c.stopped {
			break
		}
		log.Println("## serving")
		c.Serve()
	}
}

func (c *Connection) Stop() {
	c.stopped = true
	close(c.stopChan)
	if c.peer != nil {
		c.peer.Close()
	}
	if c.httpServer != nil {
		c.httpServer.Close()
	}
}
