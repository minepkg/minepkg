package remote

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

func (w *Connection) Send(name string, data any) error {
	msg := Response{Event: name, Data: data}
	return w.SendMessage(&msg)
}

func (w *Connection) SendMessage(msg *Response) error {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if w.DataChannel == nil || w.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		w.outBuffer = append(w.outBuffer, msg)
		return nil
	}

	return w.DataChannel.SendText(string(msgJSON))
}

type Message struct {
	ID    string          `json:"id,omitempty"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type Response struct {
	ID    string `json:"id,omitempty"`
	Event string `json:"event"`
	Data  any    `json:"data"`
}
