package dockerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Container struct {
	ID      string
	Name    string
	Image   string
	State   string
	Status  string
	Health  string
	Created time.Time
}

type Client struct {
	httpClient *http.Client
}

func New(socketPath string) *Client {
	transport := &http.Transport{
		DisableKeepAlives: false,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: 2 * time.Second}
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   3 * time.Second,
		},
	}
}

func (client *Client) Containers(ctx context.Context) ([]Container, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://docker/v1.41/containers/json?all=0",
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return nil, fmt.Errorf("docker API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		State   string   `json:"State"`
		Status  string   `json:"Status"`
		Created int64    `json:"Created"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(payload))
	for _, item := range payload {
		name := shortID(item.ID)
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}

		health := "unknown"
		lowerStatus := strings.ToLower(item.Status)
		switch {
		case strings.Contains(lowerStatus, "(healthy)"):
			health = "healthy"
		case strings.Contains(lowerStatus, "(unhealthy)"):
			health = "unhealthy"
		case item.State == "running":
			health = "running"
		}

		result = append(result, Container{
			ID:      item.ID,
			Name:    name,
			Image:   item.Image,
			State:   item.State,
			Status:  item.Status,
			Health:  health,
			Created: time.Unix(item.Created, 0),
		})
	}

	return result, nil
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
