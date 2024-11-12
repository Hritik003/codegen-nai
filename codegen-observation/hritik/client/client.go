// Package client contains code for making HTTP calls
package client

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"
)

// IClient is the interface for an API client.
type IClient interface {
	Do(context.Context, *http.Request) (*http.Response, []byte, error)
	MakeRequestWithRetry(ctx context.Context, url string, method string, reqBody *bytes.Buffer, headers map[string]string, maxRetries int, retryDelay time.Duration) error
	SetTimeout(timeout time.Duration)
	GetTimeout() time.Duration
}

// Client struct has http client object
type client struct {
	httpClient *http.Client
}

// NewClient returns a new http client
func NewClient() IClient {
	return &client{
		httpClient: &http.Client{},
	}
}

func (c *client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func (c *client) GetTimeout() time.Duration {
	return c.httpClient.Timeout
}

// Do makes http request to a server
func (c *client) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	var buf bytes.Buffer
	var body []byte
	_, err = buf.ReadFrom(resp.Body)
	if err == nil {
		body = buf.Bytes()
	}
	return resp, body, err
}

func (c *client) MakeRequestWithRetry(ctx context.Context, url string, method string, reqBody *bytes.Buffer, headers map[string]string, maxRetries int, retryDelay time.Duration) error {
	var resp *http.Response
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return err
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		// Call the Do function to make the HTTP request
		resp, _, err = c.Do(ctx, req)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			// Request succeeded, return
			return nil
		}
		defer func() {
			if resp != nil {
				resp.Body.Close() //nolint:errcheck,gosec
			}
		}()
		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	// If all retries failed, return the last error
	return errors.New("failed to make request after multiple retries")
}
