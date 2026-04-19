package sse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
)

// Client represents an SSE stream client with a target stream URL and the HTTP
// client used to open stream connections.
type Client struct {
	StreamURL  string
	HTTPClient *http.Client
}

// NewClient creates a Client configured for the provided SSE stream URL.
func NewClient(streamURL string) *Client {
	return &Client{
		StreamURL:  streamURL,
		HTTPClient: http.DefaultClient,
	}
}

// OpenStreamReader opens the SSE stream and returns a buffered reader and a
// closer for the underlying response body.
func (c *Client) OpenStreamReader(ctx context.Context) (*bufio.Reader, io.Closer, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.StreamURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build SSE request: %w", err)
	}
	request.Header.Set("Accept", "text/event-stream")

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to SSE stream: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		err = response.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf(
				"unexpected SSE response status: %s, failed to close response body: %w",
				response.Status, err,
			)
		}
		return nil, nil, fmt.Errorf("unexpected SSE response status: %s", response.Status)
	}

	return bufio.NewReader(response.Body), response.Body, nil
}

// NewScanner builds a scanner over an SSE stream reader.
func (c *Client) NewScanner(reader *bufio.Reader) *bufio.Scanner {
	return bufio.NewScanner(reader)
}
