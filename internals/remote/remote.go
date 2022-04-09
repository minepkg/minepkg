package remote

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type Connection struct {
	*webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
}

type WebRequest struct {
	ID string `json:"id"`
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
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	return &Connection{
		PeerConnection: peerConnection,
	}
}

func (w *Connection) ListenForHandshake(ctx context.Context) error {
	peerConnection := w.PeerConnection

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	dataChannelOpen := make(chan struct{})

	// handle incoming data
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			w.DataChannel = d

			close(dataChannelOpen)
		})
	})

	server := http.Server{Addr: ":20876"}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// only allow requests from allowed origins
		if !isOriginAllowed(origin) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "content-type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "content-type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		log.Println("Signaling request received")
		var offer webrtc.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			panic(err)
		}

		if err := peerConnection.SetRemoteDescription(offer); err != nil {
			panic(err)
		}

		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		} else if err = peerConnection.SetLocalDescription(answer); err != nil {
			panic(err)
		}

		// Block until ICE Gathering is complete, disabling trickle ICE
		// we do this because we only can exchange one signaling message
		<-gatherComplete

		response, err := json.Marshal(*peerConnection.LocalDescription())
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(response); err != nil {
			panic(err)
		}
	})

	log.Println("Listening for handshake via HTTP on :20876")
	go server.ListenAndServe()
	// Wait for the DataChannel to open
	<-dataChannelOpen
	log.Println("Data channel is open!")
	// then shutdown the server
	if err := server.Shutdown(context.Background()); err != nil {
		return err
	}
	server.Shutdown(context.Background())
	log.Println("Handshake complete, shutting down server")
	return nil
}
