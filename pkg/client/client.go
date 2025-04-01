package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents a client for the distributed cache
type Client struct {
	serverAddr string;
	httpClient *http.Client;
}

// Creates a new cache client
func NewClient (serverAddr string) (*Client) {
	return &Client{
		serverAddr: serverAddr,
		httpClient: &http.Client{},
	}
}

// Retrieves a value from the cache
func (c *Client) Get (key string) (interface{}, error) {
	url := fmt.Sprintf("%s/get?key=%s", c.serverAddr, key);

	resp, err := c.httpClient.Get(url);
	if err != nil {
		return nil, err;
	}
	defer resp.Body.Close();

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode);
	}

	var result struct {
		Key	string	`json:"key"`;
		Value interface{} `json:"value"`;
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err;
	}

	return result.Value, nil
}

// Adds a value to the cache
func (c *Client) Set (key string, value interface{}, ttl int64) error {
	url := fmt.Sprintf("%s/set", c.serverAddr);

	data := map[string]interface{} {
		"key": key,
		"value": value,
		"ttl": ttl,
	}

	jsonData, err := json.Marshal(data);
	if err != nil {
		return err;
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData));
	if err != nil {
		return err;
	}
	defer resp.Body.Close();

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode);
	}

	return nil;
}

// Removes a value from the cache
func (c *Client) Delete (key string) error {
	url := fmt.Sprintf("%s/delete?key=%s", c.serverAddr, key);

	req, err := http.NewRequest(http.MethodDelete, url, nil);
	if err != nil {
		return err;
	}

	resp, err := c.httpClient.Do(req);
	if err != nil {
		return err;
	}
	defer resp.Body.Close();

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode);
	}

	return nil;
}