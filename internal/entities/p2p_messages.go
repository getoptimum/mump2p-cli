package entities

import (
	"encoding/json"
	"io"
)

type P2PMessage struct {
	SourceNodeID   string `json:"source_node_id"`
	UpstreamPeerID string `json:"upstream_peer_id,omitempty"`
	Topic          string `json:"topic"`
	MessageID      string `json:"message_id"`
	Message        []byte `json:"message"`
}

// Marshal serializes the P2PMessage into JSON format.
func (m *P2PMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalP2PMessage deserializes JSON data into a P2PMessage struct.
func UnmarshalP2PMessage(data []byte) (*P2PMessage, error) {
	var msg P2PMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeFrom reads JSON data from an io.Reader and decodes it into the P2PMessage struct.
func (m *P2PMessage) DecodeFrom(r io.Reader) error {
	return json.NewDecoder(r).Decode(m)
}
