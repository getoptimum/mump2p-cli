package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/getoptimum/optcli/internal/config"
)

// Client handles Auth0 API interactions
type Client struct {
	domain   string
	clientID string
	audience string
	scope    string
}

// NewClient creates a new Auth0 client
func NewClient() *Client {
	cfg := config.LoadConfig()
	return &Client{
		domain:   cfg.AuthDomain,
		clientID: cfg.AuthClientID,
		audience: cfg.AuthAudience,
		scope:    "openid email offline_access",
	}
}

// Login initiates the device authorization flow
func (c *Client) Login() (*StoredToken, error) {
	// request device code
	deviceCode, err := c.requestDeviceCode()
	if err != nil {
		return nil, err
	}

	// show instructions to user
	fmt.Println("\nTo complete authentication:")
	fmt.Printf("1. Visit: %s\n", deviceCode.VerificationURIComplete)
	fmt.Printf("2. Or go to %s and enter code: %s\n", deviceCode.VerificationURI, deviceCode.UserCode)
	fmt.Printf("3. This code expires in %d minutes\n", deviceCode.ExpiresIn/60)
	fmt.Println("\nWaiting for you to complete authentication in the browser...")

	// poll for token
	token, err := c.pollForToken(deviceCode)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// requestDeviceCode starts the device authorization flow
func (c *Client) requestDeviceCode() (*DeviceCodeResponse, error) {
	payload := map[string]string{
		"client_id": c.clientID,
		"audience":  c.audience,
		"scope":     c.scope,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error creating device code request: %v", err)
	}

	// Request device code from Auth0
	resp, err := http.Post(
		fmt.Sprintf("https://%s/oauth/device/code", c.domain),
		"application/json",
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed: %s", string(body))
	}

	// parse device code response
	var deviceCode DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceCode); err != nil {
		return nil, fmt.Errorf("error parsing device code response: %v", err)
	}

	return &deviceCode, nil
}

// pollForToken continuously polls Auth0 for a token
func (c *Client) pollForToken(deviceCode *DeviceCodeResponse) (*StoredToken, error) {
	payload := map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode.DeviceCode,
		"client_id":   c.clientID,
		"scope":       c.scope,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %v", err)
	}

	// poll until we get a token or time out
	interval := time.Duration(deviceCode.Interval) * time.Second
	timeout := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for time.Now().Before(timeout) {
		// wait for the polling interval
		time.Sleep(interval)

		// token request
		resp, err := http.Post(
			fmt.Sprintf("https://%s/oauth/token", c.domain),
			"application/json",
			bytes.NewBuffer(payloadBytes),
		)
		if err != nil {
			continue // try again on network errors
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck
		if err != nil {
			continue // try again on read errors
		}

		// check for error responses
		if resp.StatusCode != http.StatusOK {
			var errorResp map[string]string
			if err := json.Unmarshal(body, &errorResp); err != nil {
				continue // try again on parse errors
			}

			// ff authorization is pending, keep polling
			if errorResp["error"] == "authorization_pending" {
				continue
			}

			// if slow_down error, increase interval
			if errorResp["error"] == "slow_down" {
				interval += time.Second
				continue
			}

			// other errors are terminal
			return nil, fmt.Errorf("token request failed: %s", errorResp["error_description"])
		}

		// parse token response
		var tokenResp TokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, fmt.Errorf("error parsing token response: %v", err)
		}

		// calculate expiration time
		expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

		// create stored token
		storedToken := &StoredToken{
			Token:        tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			ExpiresAt:    expiresAt,
		}

		return storedToken, nil
	}

	return nil, fmt.Errorf("device code expired, authentication timed out")
}

// RefreshToken obtains a new access token using the refresh token
func (c *Client) RefreshToken(refreshToken string) (*StoredToken, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     c.clientID,
		"refresh_token": refreshToken,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error creating refresh payload: %v", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("https://%s/oauth/token", c.domain),
		"application/json",
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("refresh token request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token failed (status %d): %s",
			resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("error parsing token response: %v", err)
	}

	// calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	storedToken := &StoredToken{
		Token:        tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}

	return storedToken, nil
}

// GetValidToken retrieves the stored token, refreshing if needed
func (c *Client) GetValidToken(storage *Storage) (*StoredToken, error) {
	// load existing token
	token, err := storage.LoadToken()
	if err != nil {
		return nil, err
	}

	// check if token is still valid (with 5 minute buffer)
	if time.Until(token.ExpiresAt) > 5*time.Minute {
		return token, nil
	}

	// token is expired or close to expiry, try to refresh
	if token.RefreshToken != "" {
		refreshedToken, err := c.RefreshToken(token.RefreshToken)
		if err == nil {
			// save the refreshed token
			if err := storage.SaveToken(refreshedToken); err != nil {
				return nil, fmt.Errorf("error saving refreshed token: %v", err)
			}
			return refreshedToken, nil
		}
		// log but don't fail completely - try using the original token
		fmt.Printf("Warning: Failed to refresh token: %v\n", err)
	}

	// if token is completely expired and can't be refreshed, return error
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired, please login again")
	}

	// return original token if still valid but close to expiry
	return token, nil
}
