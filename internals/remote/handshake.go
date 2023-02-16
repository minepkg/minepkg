package remote

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type handshakeError struct {
	Message string `json:"message"`
}

func (c *Connection) ListenForHandshake(ctx context.Context) error {
	dataChannelOpen := make(chan struct{})

	server := http.Server{Addr: "localhost:20876"}
	c.httpServer = &server

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			panic(err)
		}
		c.peer = peerConnection

		// handle incoming data
		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			d.OnOpen(func() {
				c.DataChannel = d

				close(dataChannelOpen)
			})
		})

		log.Println("handshake request")
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
			http.Error(w, "Bad Request", http.StatusBadRequest)
			outErr := handshakeError{Message: err.Error()}
			out, _ := json.Marshal(&outErr)
			w.Write(out)
			return
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

	go server.ListenAndServe()
	log.Println("Listening for handshake via HTTP on localhost:20876")
	// Wait for the DataChannel to open
	select {
	case <-dataChannelOpen:
	case <-c.stopChan:
		return nil
	}
	log.Println("Data channel is open!")

	// send buffered messages
	for _, msg := range c.outBuffer {
		c.SendMessage(msg)
	}
	c.outBuffer = []*Response{}

	// then shutdown the server
	if err := server.Shutdown(context.Background()); err != nil {
		return err
	}
	server.Shutdown(context.Background())
	log.Println("Handshake complete, shutting down server")
	return nil
}
