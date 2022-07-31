package remote

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/webrtc/v3"
)

type Handler interface {
	Handle(c *Message) *Response
}

type handlerFunc struct {
	f func(*Message) *Response
}

func (h handlerFunc) Handle(e *Message) *Response {
	return h.f(e)
}

func (w *Connection) HandleFunc(eventName string, handler func(*Message) *Response) {
	fmt.Println("added handler for", eventName)
	w.handlers[eventName] = handlerFunc{handler}
}

func (w *Connection) Serve() {
	go w.Send("ready", nil)

	w.DataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		var request Message
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			panic(err)
		}

		if request.Event == "bye" {
			log.Println("RECEIVED BYE 👋")
			w.peer.Close()
			return
		}

		if request.Event == "ping" {
			// send as response
			if request.ID != "" {
				w.SendMessage(&Response{ID: request.ID, Event: "ping"})
				return
			}
			w.Send("pong", nil)
			return
		}

		if handler, ok := w.handlers[request.Event]; ok {
			log.Println("handling", request.Event)
			response := handler.Handle(&request)

			if response != nil {
				response.ID = request.ID

				if response.Event == "" {
					response.Event = request.Event
				}
				w.SendMessage(response)
			} else {
				w.SendMessage(&Response{ID: request.ID, Event: request.Event})
			}
			return
		}
		log.Printf("unhandled event: %s", request.Event)
	})

	// w.DataChannel.OnClose(closeChannel)

	// w.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
	// 	if connectionState == webrtc.ICEConnectionStateDisconnected {
	// 		closeChannel()
	// 	}
	// })
	log.Println("Stopping remote serve")
}
