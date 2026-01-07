package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the default Docker Hub API base URL
	DefaultBaseURL = "https://hub.docker.com/v2"
	// DefaultPageSize is the default page size for API requests
	DefaultPageSize = 100
)

// Client represents a Docker Hub API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	username   string
	limiter    *rate.Limiter
}

// NewClient creates a new Docker Hub API client
func NewClient() *Client {
	return &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Every(time.Second), 5), // 5 requests per second
	}
}

// Authenticate authenticates with Docker Hub using username and password
func (c *Client) Authenticate(ctx context.Context, username, password string) error {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/users/login/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return NewAPIError(resp.StatusCode, "/users/login/", string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.token = loginResp.Token
	c.username = username
	return nil
}

// AuthenticateWithToken authenticates using a personal access token
func (c *Client) AuthenticateWithToken(token string) {
	c.token = token
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Add authorization header if token is available
	if c.token != "" {
		req.Header.Set("Authorization", "JWT "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNetworkError, err)
	}

	// Handle rate limiting with exponential backoff
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()

		// Exponential backoff: try up to 5 times
		for i := 0; i < 5; i++ {
			wait := time.Duration(1<<uint(i)) * time.Second // 1s, 2s, 4s, 8s, 16s
			time.Sleep(wait)

			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrNetworkError, err)
			}

			if resp.StatusCode != http.StatusTooManyRequests {
				return resp, nil
			}
			resp.Body.Close()
		}

		return nil, ErrRateLimited
	}

	return resp, nil
}

// ListTags fetches all tags for a repository
func (c *Client) ListTags(ctx context.Context, repo string) ([]Tag, error) {
	var allTags []Tag
	page := 1

	for {
		url := fmt.Sprintf("%s/repositories/%s/tags/?page=%d&page_size=%d", c.baseURL, repo, page, DefaultPageSize)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.doRequest(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, ErrNotFound
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, ErrUnauthorized
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, NewAPIError(resp.StatusCode, url, string(bodyBytes))
		}

		var tagsResp TagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode tags response: %w", err)
		}
		resp.Body.Close()

		allTags = append(allTags, tagsResp.Results...)

		// Check if there are more pages
		if tagsResp.Next == nil || *tagsResp.Next == "" {
			break
		}

		page++
	}

	return allTags, nil
}

// DeleteTag deletes a specific tag from a repository
func (c *Client) DeleteTag(ctx context.Context, repo, tag string) error {
	url := fmt.Sprintf("%s/repositories/%s/tags/%s/", c.baseURL, repo, tag)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return NewAPIError(resp.StatusCode, url, string(bodyBytes))
	}

	return nil
}

// GetRepository fetches repository information
func (c *Client) GetRepository(ctx context.Context, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s/", c.baseURL, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, NewAPIError(resp.StatusCode, url, string(bodyBytes))
	}

	var repository Repository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("failed to decode repository response: %w", err)
	}

	return &repository, nil
}
