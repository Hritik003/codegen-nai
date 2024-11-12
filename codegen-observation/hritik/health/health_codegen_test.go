package client

import (
	"context"
	"net/http"
	"testing"
)

func TestNewHealthClient(t *testing.T) {
	client := &IClient{}
	ic := NewHealthClient(client)
	_, ok := ic.(*healthClient)
	require.True(t, ok)
}

func TestCheckHealth(t *testing.T) {
	client := &IClient{}
	ic := NewHealthClient(client)
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	resp, _, err := ic.client.Do(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCheckHealthInternal(t *testing.T) {
	client := &IClient{}
	ic := NewHealthClient(client)
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	resp, _, err := ic.client.Do(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCheckHealthInternalError(t *testing.T) {
	client := &IClient{}
	ic := NewHealthClient(client)
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	resp, _, err := ic.client.Do(context.Background(), req)
	require.Error(t, err)
	defer resp.Body.Close()
}

func TestCheckHealthInternalUnknownStatusCode(t *testing.T) {
	client := &IClient{}
	ic := NewHealthClient(client)
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	resp, _, err := ic.client.Do(context.Background(), req)
	require.Error(t, err)
	defer resp.Body.Close()
}