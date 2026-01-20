package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Common errors for resource operations
var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("unauthorized - session may have expired")
	ErrBadRequest    = errors.New("bad request")
	ErrInternalError = errors.New("internal server error")
)

// ClientConfig holds configuration for the DreamFactory client
type ClientConfig struct {
	// BaseURL is the DreamFactory instance URL
	BaseURL string
	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration
	// InsecureSkipVerify disables TLS certificate verification (not recommended for production)
	InsecureSkipVerify bool
	// TLSMinVersion sets the minimum TLS version (default: TLS 1.2)
	TLSMinVersion uint16
}

// Client represents a DreamFactory API client
type Client struct {
	baseURL      string
	sessionToken string
	httpClient   *http.Client
}

// NewClient creates a new DreamFactory API client with default settings
func NewClient(baseURL string) *Client {
	return NewClientWithConfig(ClientConfig{
		BaseURL: baseURL,
	})
}

// NewClientWithConfig creates a new DreamFactory API client with custom configuration
func NewClientWithConfig(config ClientConfig) *Client {
	// Normalize base URL (remove trailing slashes)
	baseURL := strings.TrimRight(config.BaseURL, "/")

	// Set defaults
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	tlsMinVersion := config.TLSMinVersion
	if tlsMinVersion == 0 {
		tlsMinVersion = tls.VersionTLS12
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion:         tlsMinVersion,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	SessionToken string `json:"session_token"`
	ID           int    `json:"id"`
	Name         string `json:"name"`
	IsSysAdmin   bool   `json:"is_sys_admin"`
	IsRootAdmin  bool   `json:"is_root_admin"`
}

// Login authenticates with DreamFactory and stores the session token
func (c *Client) Login(email, password string) error {
	return c.LoginWithContext(context.Background(), email, password)
}

// LoginWithContext authenticates with DreamFactory with context support
func (c *Client) LoginWithContext(ctx context.Context, email, password string) error {
	payload := LoginRequest{
		Email:    email,
		Password: password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v2/system/admin/session", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("login failed with status %d (unable to read response body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.sessionToken = loginResp.SessionToken
	return nil
}

// doRequest performs an authenticated API request with context
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	// Validate and sanitize path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fullURL := c.baseURL + path
	if _, err := url.Parse(fullURL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DreamFactory-Session-Token", c.sessionToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	return resp, nil
}

// doRequestLegacy performs an authenticated API request without context (for backward compatibility)
func (c *Client) doRequestLegacy(method, path string, body interface{}) (*http.Response, error) {
	return c.doRequest(context.Background(), method, path, body)
}

// readErrorBody safely reads the response body for error messages
func readErrorBody(resp *http.Response) string {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "(unable to read response body)"
	}
	return string(bodyBytes)
}

// mapStatusToError converts HTTP status codes to sentinel errors
func mapStatusToError(statusCode int, body string) error {
	switch statusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, body)
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadRequest, body)
	case http.StatusInternalServerError:
		return fmt.Errorf("%w: %s", ErrInternalError, body)
	default:
		return fmt.Errorf("request failed with status %d: %s", statusCode, body)
	}
}

// ServiceConfig is a custom type that handles DreamFactory's config field
// which can be either an empty array [] or an object {}
type ServiceConfig map[string]interface{}

// UnmarshalJSON implements custom unmarshaling for ServiceConfig
func (sc *ServiceConfig) UnmarshalJSON(data []byte) error {
	// Trim whitespace
	data = bytes.TrimSpace(data)

	// Check if it's null or empty
	if len(data) == 0 || string(data) == "null" {
		*sc = make(map[string]interface{})
		return nil
	}

	// Check if it starts with '[' (array) - DreamFactory returns [] for empty config
	if len(data) > 0 && data[0] == '[' {
		*sc = make(map[string]interface{})
		return nil
	}

	// Otherwise unmarshal as a map
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal service config: %w", err)
	}
	*sc = m
	return nil
}

// Service represents a DreamFactory service
type Service struct {
	ID          int           `json:"id,omitempty"`
	Name        string        `json:"name"`
	Label       string        `json:"label,omitempty"`
	Description string        `json:"description,omitempty"`
	Type        string        `json:"type"`
	IsActive    bool          `json:"is_active"`
	Config      ServiceConfig `json:"config,omitempty"`
}

// ServiceListResponse represents the response from listing services
type ServiceListResponse struct {
	Resource []Service `json:"resource"`
}

// ServiceCreateRequest represents the request to create services
type ServiceCreateRequest struct {
	Resource []Service `json:"resource"`
}

// ServiceCreateResponse represents the response from creating services
type ServiceCreateResponse struct {
	Resource []struct {
		ID int `json:"id"`
	} `json:"resource"`
}

// ListServices returns all services
func (c *Client) ListServices() ([]Service, error) {
	return c.ListServicesWithContext(context.Background())
}

// ListServicesWithContext returns all services with context support
func (c *Client) ListServicesWithContext(ctx context.Context) ([]Service, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v2/system/service", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var listResp ServiceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode services response: %w", err)
	}

	return listResp.Resource, nil
}

// GetService returns a service by ID
func (c *Client) GetService(id int) (*Service, error) {
	return c.GetServiceWithContext(context.Background(), id)
}

// GetServiceWithContext returns a service by ID with context support
func (c *Client) GetServiceWithContext(ctx context.Context, id int) (*Service, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v2/system/service/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var service Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("failed to decode service response: %w", err)
	}

	return &service, nil
}

// CreateService creates a new service
func (c *Client) CreateService(service Service) (int, error) {
	return c.CreateServiceWithContext(context.Background(), service)
}

// CreateServiceWithContext creates a new service with context support
func (c *Client) CreateServiceWithContext(ctx context.Context, service Service) (int, error) {
	payload := ServiceCreateRequest{
		Resource: []Service{service},
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v2/system/service", payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var createResp ServiceCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return 0, fmt.Errorf("failed to decode create response: %w", err)
	}

	if len(createResp.Resource) == 0 {
		return 0, fmt.Errorf("no service ID returned in response")
	}

	return createResp.Resource[0].ID, nil
}

// UpdateService updates an existing service
func (c *Client) UpdateService(id int, service Service) error {
	return c.UpdateServiceWithContext(context.Background(), id, service)
}

// UpdateServiceWithContext updates an existing service with context support
func (c *Client) UpdateServiceWithContext(ctx context.Context, id int, service Service) error {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/v2/system/service/%d", id), service)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// DeleteService deletes a service by ID
func (c *Client) DeleteService(id int) error {
	return c.DeleteServiceWithContext(context.Background(), id)
}

// DeleteServiceWithContext deletes a service by ID with context support
func (c *Client) DeleteServiceWithContext(ctx context.Context, id int) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v2/system/service/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// Role represents a DreamFactory role
type Role struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
}

// RoleListResponse represents the response from listing roles
type RoleListResponse struct {
	Resource []Role `json:"resource"`
}

// ListRoles returns all roles
func (c *Client) ListRoles() ([]Role, error) {
	return c.ListRolesWithContext(context.Background())
}

// ListRolesWithContext returns all roles with context support
func (c *Client) ListRolesWithContext(ctx context.Context) ([]Role, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v2/system/role", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var listResp RoleListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode roles response: %w", err)
	}

	return listResp.Resource, nil
}

// GetRole returns a role by ID
func (c *Client) GetRole(id int) (*Role, error) {
	return c.GetRoleWithContext(context.Background(), id)
}

// GetRoleWithContext returns a role by ID with context support
func (c *Client) GetRoleWithContext(ctx context.Context, id int) (*Role, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v2/system/role/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var role Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode role response: %w", err)
	}

	return &role, nil
}

// CreateRole creates a new role
func (c *Client) CreateRole(role Role) (int, error) {
	return c.CreateRoleWithContext(context.Background(), role)
}

// CreateRoleWithContext creates a new role with context support
func (c *Client) CreateRoleWithContext(ctx context.Context, role Role) (int, error) {
	payload := struct {
		Resource []Role `json:"resource"`
	}{
		Resource: []Role{role},
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v2/system/role", payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var createResp struct {
		Resource []struct {
			ID int `json:"id"`
		} `json:"resource"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return 0, fmt.Errorf("failed to decode create response: %w", err)
	}

	if len(createResp.Resource) == 0 {
		return 0, fmt.Errorf("no role ID returned in response")
	}

	return createResp.Resource[0].ID, nil
}

// UpdateRole updates an existing role
func (c *Client) UpdateRole(id int, role Role) error {
	return c.UpdateRoleWithContext(context.Background(), id, role)
}

// UpdateRoleWithContext updates an existing role with context support
func (c *Client) UpdateRoleWithContext(ctx context.Context, id int, role Role) error {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/v2/system/role/%d", id), role)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// DeleteRole deletes a role by ID
func (c *Client) DeleteRole(id int) error {
	return c.DeleteRoleWithContext(context.Background(), id)
}

// DeleteRoleWithContext deletes a role by ID with context support
func (c *Client) DeleteRoleWithContext(ctx context.Context, id int) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v2/system/role/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// App represents a DreamFactory app (API key)
type App struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	APIKey      string `json:"api_key,omitempty"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	Type        int    `json:"type"`
	RoleID      *int   `json:"role_id,omitempty"`
}

// AppListResponse represents the response from listing apps
type AppListResponse struct {
	Resource []App `json:"resource"`
}

// ListApps returns all apps
func (c *Client) ListApps() ([]App, error) {
	return c.ListAppsWithContext(context.Background())
}

// ListAppsWithContext returns all apps with context support
func (c *Client) ListAppsWithContext(ctx context.Context) ([]App, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v2/system/app", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var listResp AppListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode apps response: %w", err)
	}

	return listResp.Resource, nil
}

// GetApp returns an app by ID
func (c *Client) GetApp(id int) (*App, error) {
	return c.GetAppWithContext(context.Background(), id)
}

// GetAppWithContext returns an app by ID with context support
func (c *Client) GetAppWithContext(ctx context.Context, id int) (*App, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v2/system/app/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var app App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode app response: %w", err)
	}

	return &app, nil
}

// CreateApp creates a new app and fetches the generated API key
func (c *Client) CreateApp(app App) (int, string, error) {
	return c.CreateAppWithContext(context.Background(), app)
}

// CreateAppWithContext creates a new app with context support
func (c *Client) CreateAppWithContext(ctx context.Context, app App) (int, string, error) {
	payload := struct {
		Resource []App `json:"resource"`
	}{
		Resource: []App{app},
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v2/system/app", payload)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, "", mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	var createResp struct {
		Resource []struct {
			ID int `json:"id"`
		} `json:"resource"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return 0, "", fmt.Errorf("failed to decode create response: %w", err)
	}

	if len(createResp.Resource) == 0 {
		return 0, "", fmt.Errorf("no app ID returned in response")
	}

	id := createResp.Resource[0].ID

	// Fetch the created app to get the API key (DreamFactory doesn't return it on create)
	createdApp, err := c.GetAppWithContext(ctx, id)
	if err != nil {
		return id, "", fmt.Errorf("app created but failed to fetch API key: %w", err)
	}

	return id, createdApp.APIKey, nil
}

// UpdateApp updates an existing app
func (c *Client) UpdateApp(id int, app App) error {
	return c.UpdateAppWithContext(context.Background(), id, app)
}

// UpdateAppWithContext updates an existing app with context support
func (c *Client) UpdateAppWithContext(ctx context.Context, id int, app App) error {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/v2/system/app/%d", id), app)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// DeleteApp deletes an app by ID
func (c *Client) DeleteApp(id int) error {
	return c.DeleteAppWithContext(context.Background(), id)
}

// DeleteAppWithContext deletes an app by ID with context support
func (c *Client) DeleteAppWithContext(ctx context.Context, id int) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v2/system/app/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatusToError(resp.StatusCode, readErrorBody(resp))
	}

	return nil
}
