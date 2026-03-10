package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type CachedSession struct {
	ProxyURL     string   `json:"proxy_url"`
	ClientID     string   `json:"client_id"`
	Topics       []string `json:"topics"`
	Capabilities []string `json:"capabilities"`
	ExposeAmount uint32   `json:"expose_amount"`
	Session      Session  `json:"session"`
}

func sessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mump2p")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func cachePath() (string, error) {
	dir, err := sessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func lockPath() (string, error) {
	dir, err := sessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.lock"), nil
}

func acquireLock() (*os.File, error) {
	p, err := lockPath()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func releaseLock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck
	f.Close()
}

func loadCached() (*CachedSession, error) {
	p, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c CachedSession
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveCached(c *CachedSession) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

func sortedKey(s []string) string {
	cp := make([]string, len(s))
	copy(cp, s)
	sort.Strings(cp)
	return strings.Join(cp, ",")
}

func (c *CachedSession) matches(proxyURL, clientID string, topics, capabilities []string) bool {
	if c.ProxyURL != proxyURL || c.ClientID != clientID {
		return false
	}
	if sortedKey(c.Capabilities) != sortedKey(capabilities) {
		return false
	}
	cached := make(map[string]bool, len(c.Topics))
	for _, t := range c.Topics {
		cached[t] = true
	}
	for _, t := range topics {
		if !cached[t] {
			return false
		}
	}
	return true
}

func (c *CachedSession) needsRefresh() bool {
	ra, err := time.Parse(time.RFC3339, c.Session.RefreshAfter)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(ra)
}

func (c *CachedSession) isExpired() bool {
	ea, err := time.Parse(time.RFC3339, c.Session.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(ea)
}

func isUsable(cached *CachedSession, proxyURL, clientID string, topics, capabilities []string) bool {
	return cached != nil &&
		cached.matches(proxyURL, clientID, topics, capabilities) &&
		!cached.isExpired() &&
		!cached.needsRefresh()
}

// GetOrCreateSession returns a cached session if valid, refreshes if past
// the refresh window, or creates a new session. Uses a file lock to prevent
// concurrent processes from each creating separate sessions.
func GetOrCreateSession(proxyURL, clientID string, topics, capabilities []string, exposeAmount uint32) (*Session, bool, error) {
	// Fast path: read without lock — if valid, return immediately.
	if cached, err := loadCached(); err == nil && isUsable(cached, proxyURL, clientID, topics, capabilities) {
		return &cached.Session, true, nil
	}

	// Slow path: acquire lock, re-check (another process may have refreshed).
	lf, lockErr := acquireLock()
	if lockErr != nil {
		// If locking fails, fall through to create without cache.
		sess, err := CreateSession(proxyURL, clientID, topics, capabilities, exposeAmount)
		return sess, false, err
	}
	defer releaseLock(lf)

	if cached, err := loadCached(); err == nil && isUsable(cached, proxyURL, clientID, topics, capabilities) {
		return &cached.Session, true, nil
	}

	sess, err := CreateSession(proxyURL, clientID, topics, capabilities, exposeAmount)
	if err != nil {
		return nil, false, err
	}

	c := &CachedSession{
		ProxyURL:     proxyURL,
		ClientID:     clientID,
		Topics:       topics,
		Capabilities: capabilities,
		ExposeAmount: exposeAmount,
		Session:      *sess,
	}
	if saveErr := saveCached(c); saveErr != nil {
		fmt.Printf("Warning: could not cache session: %v\n", saveErr)
	}

	return sess, false, nil
}

// InvalidateSession removes the cached session file.
func InvalidateSession() {
	p, err := cachePath()
	if err != nil {
		return
	}
	os.Remove(p)
}
