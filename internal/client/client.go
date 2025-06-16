package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	fb "terraform-provider-flashblade/fb_sdk"
)

type Client struct {
	*fb.ClientWithResponses
}

func New(endpoint, apiToken string, insecure bool) (*Client, error) {
	ctx := context.Background()
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if insecure {
		tflog.Warn(ctx, "TLS certificate verification is disabled.")
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	loginClient := &http.Client{Timeout: 30 * time.Second, Transport: transport}

	tflog.Debug(ctx, "Attempting to log in to get session token...")
	loginURL := endpoint + "/api/login"
	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("api-token", apiToken)
	resp, err := loginClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API login failed with status %s", resp.Status)
	}
	sessionToken := resp.Header.Get("x-auth-token")
	if sessionToken == "" {
		return nil, fmt.Errorf("API login succeeded but did not return an x-auth-token header")
	}
	tflog.Debug(ctx, "Successfully obtained session token.")

	apiClient := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	clientWithResponses, err := fb.NewClientWithResponses(endpoint, fb.WithHTTPClient(apiClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create fb sdk client with responses: %w", err)
	}
	sessionAuthEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-auth-token", sessionToken)
		return nil
	}
	clientWithResponses.ClientInterface.(*fb.Client).RequestEditors = append(
		clientWithResponses.ClientInterface.(*fb.Client).RequestEditors,
		sessionAuthEditor,
	)
	return &Client{ClientWithResponses: clientWithResponses}, nil
}

func newApiError(op string, resp *http.Response, body []byte) error {
	return fmt.Errorf("API Error during %s: status %s, body: %s", op, resp.Status, string(body))
}

func (c *Client) GetFileSystemByName(ctx context.Context, name string) (*fb.FileSystem, error) {
	params := &fb.GetApi217FileSystemsParams{Names: &[]string{name}}
	resp, err := c.GetApi217FileSystemsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get file system: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newApiError("GetFileSystem", resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil || resp.JSON200.Items == nil || len(*resp.JSON200.Items) == 0 {
		return nil, nil
	}
	return &(*resp.JSON200.Items)[0], nil
}

func (c *Client) CreateFileSystem(ctx context.Context, name string, fs *fb.FileSystemPost) (*fb.FileSystem, error) {
	params := &fb.PostApi217FileSystemsParams{Names: []string{name}}
	resp, err := c.PostApi217FileSystemsWithResponse(ctx, params, *fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create file system: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newApiError("CreateFileSystem", resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil || resp.JSON200.Items == nil || len(*resp.JSON200.Items) == 0 {
		return nil, fmt.Errorf("API did not return created file system in response")
	}
	return &(*resp.JSON200.Items)[0], nil
}

func (c *Client) UpdateFileSystem(ctx context.Context, name string, fs *fb.FileSystemPatch) (*fb.FileSystem, error) {
	params := &fb.PatchApi217FileSystemsParams{Names: &[]string{name}}
	resp, err := c.PatchApi217FileSystemsWithResponse(ctx, params, *fs)
	if err != nil {
		return nil, fmt.Errorf("failed to update file system: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newApiError("UpdateFileSystem", resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil || resp.JSON200.Items == nil || len(*resp.JSON200.Items) == 0 {
		return nil, fmt.Errorf("API did not return updated file system in response")
	}
	return &(*resp.JSON200.Items)[0], nil
}

func (c *Client) EradicateFileSystem(ctx context.Context, name string) error {
	params := &fb.DeleteApi217FileSystemsParams{Names: &[]string{name}}
	resp, err := c.DeleteApi217FileSystemsWithResponse(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to eradicate file system: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newApiError("EradicateFileSystem", resp.HTTPResponse, resp.Body)
	}
	return nil
}
