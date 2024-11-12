package client

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

var (
	client = NewClient()
)

func TestNewClient(t *testing.T) {
	if client == nil {
		t.Errorf("NewClient returned nil")
	}
}

func TestSetTimeout(t *testing.T) {
	client.SetTimeout(10 * time.Second)
	if client.GetTimeout() != 10*time.Second {
		t.Errorf("SetTimeout did not set timeout correctly")
	}
}

func TestGetTimeout(t *testing.T) {
	client.SetTimeout(10 * time.Second)
	if client.GetTimeout() != 10*time.Second {
		t.Errorf("GetTimeout did not return set timeout")
	}
}

func TestDo(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Errorf("Error creating request: %v", err)
	}
	resp, body, err := client.Do(context.Background(), req)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}
	if resp == nil || len(body) == 0 {
		t.Errorf("Response or body is empty")
	}
}

func TestMakeRequestWithRetry(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Errorf("Error creating request: %v", err)
	}
	err = client.MakeRequestWithRetry(context.Background(), "https://example.com", "GET", nil, map[string]string{}, 3, 1*time.Second)
	if err != nil {
		t.Errorf("Error making request with retry: %v", err)
	}
}