// oura/client.go
package oura

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const baseURL = "https://api.ouraring.com/v2/usercollection"

type Client struct {
	accessToken string
	http        *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{accessToken: accessToken, http: &http.Client{}}
}

func (c *Client) SetAccessToken(token string) { c.accessToken = token }

type PagedResponse[T any] struct {
	Data []T `json:"data"`
}

func get[T any](ctx context.Context, c *Client, path string, params url.Values) ([]T, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oura %s %d: %s", path, resp.StatusCode, body)
	}

	var pr PagedResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return pr.Data, nil
}
