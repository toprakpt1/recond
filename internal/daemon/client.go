package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func SocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".recond", "recond.sock")
}

func SendCommand(action string, payload interface{}) (*Response, error) {
	conn, err := net.Dial("unix", SocketPath())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon (is it running?): %w", err)
	}
	defer conn.Close()

	var payloadJSON json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		payloadJSON = data
	}

	req := Request{
		Action:  action,
		Payload: payloadJSON,
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}
