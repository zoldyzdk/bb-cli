package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	httpClient *http.Client
	username   string
	token      string
}

func NewClient(username, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   username,
		token:      token,
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr models.APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	if target != nil {
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (c *Client) Get(path string, target interface{}) error {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, target)
}

func (c *Client) Post(path string, body interface{}, target interface{}) error {
	req, err := c.newRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, target)
}

func (c *Client) GetRaw(path string) (string, error) {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return string(data), nil
}

type CurrentUser struct {
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
}

func (c *Client) GetCurrentUser() (*CurrentUser, error) {
	var user CurrentUser
	if err := c.Get("/user", &user); err != nil {
		return nil, err
	}
	return &user, nil
}
