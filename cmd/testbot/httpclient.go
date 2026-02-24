package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient provides HTTP helpers for interacting with the server.
type HTTPClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *HTTPClient) postJSON(path, jwt string, payload, result interface{}) error {
	return c.doJSON(http.MethodPost, path, jwt, payload, result)
}

func (c *HTTPClient) getJSON(path, jwt string, result interface{}) error {
	return c.doJSON(http.MethodGet, path, jwt, nil, result)
}

func (c *HTTPClient) doJSON(method, path, jwt string, payload, result interface{}) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

type loginResponse struct {
	JWT    string `json:"jwt"`
	Player struct {
		ID int64 `json:"id"`
	} `json:"player"`
}

// Login authenticates a player and returns their JWT and player ID.
func (c *HTTPClient) Login(email, password string) (string, int64, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	var resp loginResponse
	if err := c.postJSON("/player/auth", "", payload, &resp); err != nil {
		return "", 0, fmt.Errorf("login: %w", err)
	}
	return resp.JWT, resp.Player.ID, nil
}

type createTestPlayerResponse struct {
	PlayerID int64  `json:"playerId"`
	Email    string `json:"email"`
}

// CreateTestPlayer creates a pre-verified test player via the admin endpoint.
func (c *HTTPClient) CreateTestPlayer(adminJWT, displayName, email, password string) (int64, error) {
	payload := map[string]string{
		"displayName": displayName,
		"email":       email,
		"password":    password,
	}
	var resp createTestPlayerResponse
	if err := c.postJSON("/admin/test-player", adminJWT, payload, &resp); err != nil {
		return 0, fmt.Errorf("create test player: %w", err)
	}
	return resp.PlayerID, nil
}

type createTableResponse struct {
	UUID string `json:"uuid"`
}

// CreateTable creates a new table and returns its UUID.
func (c *HTTPClient) CreateTable(jwt, name string) (string, error) {
	payload := map[string]string{"name": name}
	var resp createTableResponse
	if err := c.postJSON("/table", jwt, payload, &resp); err != nil {
		return "", fmt.Errorf("create table: %w", err)
	}
	return resp.UUID, nil
}

// TablePlayer represents a player seated at a table.
type TablePlayer struct {
	PlayerID int64 `json:"playerId"`
	Player   struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"player"`
	Active bool `json:"active"`
}

type getTableResponse struct {
	UUID    string        `json:"uuid"`
	Players []TablePlayer `json:"players"`
}

// GetTablePlayers fetches the player list for a table.
func (c *HTTPClient) GetTablePlayers(jwt, tableUUID string) ([]TablePlayer, error) {
	var resp getTableResponse
	if err := c.getJSON("/table/"+tableUUID, jwt, &resp); err != nil {
		return nil, fmt.Errorf("get table: %w", err)
	}
	return resp.Players, nil
}

// SetPlayerPassword resets a player's password via the admin endpoint.
func (c *HTTPClient) SetPlayerPassword(adminJWT string, playerID int64, newPassword string) error {
	payload := map[string]interface{}{
		"key":   "password",
		"value": newPassword,
	}
	path := fmt.Sprintf("/admin/player/%d", playerID)
	if err := c.postJSON(path, adminJWT, payload, nil); err != nil {
		return fmt.Errorf("set player password: %w", err)
	}
	return nil
}

// JoinTable joins a player to a table.
func (c *HTTPClient) JoinTable(jwt, tableUUID string) error {
	if err := c.postJSON("/table/"+tableUUID+"/seat", jwt, nil, nil); err != nil {
		return fmt.Errorf("join table: %w", err)
	}
	return nil
}
