package remote

import (
	"context"
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

func (w *Connection) ReceiveChannel() <-chan WebEvent {
	channel := make(chan WebEvent)
	w.DataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		var event WebEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			panic(err)
		}

		if event.Event == "bye" {
			w.handleDisconnect()
		}

		channel <- event
	})
	return channel
}

func (w *Connection) WaitFor(ctx context.Context, eventName string) WebEvent {
	for {
		select {
		case event := <-w.ReceiveChannel():
			if event.Event == eventName {
				return event
			}
		}
	}
}
