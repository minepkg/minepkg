package remote

import "encoding/json"

func (w *Connection) SendEvent(name string, data interface{}) error {
	event := WebEvent{Event: name, Data: data}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return w.DataChannel.SendText(string(eventJSON))
}

type WebEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}
