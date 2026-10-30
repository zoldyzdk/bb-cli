package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	httpClient *http.Client
	username   string
	token      string
	baseURL    string
	bearer     bool
	ctx        context.Context
}

func NewClient(username, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   username,
		token:      token,
		baseURL:    baseURL,
	}
}

// NewRepositoryClient is read-only and never falls back to personal credentials.
func NewRepositoryClient(token string) *Client {
	c := NewClient("", token)
	c.bearer = true
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many API redirects")
		}
		if req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host {
			return fmt.Errorf("API redirect outside configured origin refused")
		}
		return nil
	}
	return c
}

// WithEndpoint permits an explicit alternate API endpoint, e.g. a local API simulator.
func (c *Client) WithEndpoint(endpoint string) *Client {
	c.baseURL = strings.TrimRight(endpoint, "/")
	return c
}
func (c *Client) WithContext(ctx context.Context) *Client { c.ctx = ctx; return c }

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	if c.bearer && method != http.MethodGet {
		return nil, fmt.Errorf("repository review client only permits GET")
	}
	endpoint := c.baseURL
	if endpoint == "" {
		endpoint = baseURL
	}
	requestURL := endpoint + path
	if parsed, err := url.Parse(path); err != nil {
		return nil, err
	} else if parsed.IsAbs() {
		base, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != base.Scheme || parsed.Host != base.Host || !strings.HasPrefix(parsed.Path, base.Path+"/") {
			return nil, fmt.Errorf("API link outside configured origin refused")
		}
		requestURL = path
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.bearer {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else {
		req.SetBasicAuth(c.username, c.token)
	}
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
		if c.bearer {
			return repositoryHTTPError(resp)
		}
		var apiErr models.APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error.Message != "" {
			return &HTTPError{StatusCode: resp.StatusCode, Message: apiErr.Error.Message}
		}
		return &HTTPError{StatusCode: resp.StatusCode, Message: string(data)}
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
		if c.bearer {
			return "", repositoryHTTPError(resp)
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return string(data), nil
}

func repositoryHTTPError(resp *http.Response) *HTTPError {
	err := &HTTPError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	if seconds, parseErr := strconv.Atoi(resp.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
		err.RetryAfter = time.Duration(seconds) * time.Second
	} else if date, parseErr := http.ParseTime(resp.Header.Get("Retry-After")); parseErr == nil {
		err.RetryAfter = time.Until(date)
	}
	return err
}

type CurrentUser struct {
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
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
