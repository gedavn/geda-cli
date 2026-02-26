package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type APIError struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (e *APIError) Error() string {
	if message, ok := e.Body["message"].(string); ok && message != "" {
		return message
	}

	if e.Raw != "" {
		return e.Raw
	}

	return fmt.Sprintf("request failed with status %d", e.Status)
}

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func New(baseURL string, accessToken string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Get(p string) (map[string]any, error) {
	return c.do(http.MethodGet, p, nil)
}

func (c *Client) Post(p string, payload any) (map[string]any, error) {
	return c.do(http.MethodPost, p, payload)
}

func (c *Client) Put(p string, payload any) (map[string]any, error) {
	return c.do(http.MethodPut, p, payload)
}

func (c *Client) Delete(p string) (map[string]any, error) {
	return c.do(http.MethodDelete, p, nil)
}

func (c *Client) PostMultipartFile(p string, fileField string, filePath string, fields map[string]string) (map[string]any, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return c.doRaw(http.MethodPost, p, &body, writer.FormDataContentType())
}

func (c *Client) do(method string, p string, payload any) (map[string]any, error) {
	var bodyReader io.Reader
	contentType := ""
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		bodyReader = bytes.NewReader(payloadBytes)
		contentType = "application/json"
	}

	return c.doRaw(method, p, bodyReader, contentType)
}

func (c *Client) doRaw(method string, p string, bodyReader io.Reader, contentType string) (map[string]any, error) {
	if c.baseURL == "" {
		return nil, errors.New("base URL is required")
	}

	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	endpointURL, err := url.Parse(p)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(baseURL.Path, endpointURL.Path)
	baseURL.RawQuery = endpointURL.RawQuery

	req, err := http.NewRequest(method, baseURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			if resp.StatusCode >= 400 {
				return nil, &APIError{Status: resp.StatusCode, Raw: string(body)}
			}

			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{Status: resp.StatusCode, Body: result, Raw: string(body)}
	}

	return result, nil
}
