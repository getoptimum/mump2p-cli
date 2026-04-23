package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Node struct {
	ID        string  `json:"id"`
	Address   string  `json:"address"`
	Transport string  `json:"transport"`
	Region    string  `json:"region"`
	Ticket    string  `json:"ticket"`
	Score     float32 `json:"score"`
}

type Session struct {
	SessionID    string `json:"session_id"`
	Nodes        []Node `json:"nodes"`
	ExpiresAt    string `json:"expires_at"`
	RefreshAfter string `json:"refresh_after"`
	Error        string `json:"error,omitempty"`
}

type sessionRequest struct {
	ClientID     string   `json:"client_id"`
	Topics       []string `json:"topics"`
	Capabilities []string `json:"capabilities"`
	ExposeAmount uint32   `json:"expose_amount"`
}

func CreateSession(proxyURL, clientID, accessToken string, topics, capabilities []string, exposeAmount uint32) (*Session, error) {
	reqData := sessionRequest{
		ClientID:     clientID,
		Topics:       topics,
		Capabilities: capabilities,
		ExposeAmount: exposeAmount,
	}

	body, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session request: %w", err)
	}

	url := proxyURL + "/api/v1/session"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var sess Session
	if err := json.Unmarshal(respBody, &sess); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	if sess.Error != "" {
		return nil, fmt.Errorf("session error: %s", sess.Error)
	}

	if len(sess.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	return &sess, nil
}
