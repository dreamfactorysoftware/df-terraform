package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://example.com")
	if client.baseURL != "https://example.com" {
		t.Errorf("Expected baseURL to be 'https://example.com', got '%s'", client.baseURL)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         ClientConfig
		expectedURL    string
		expectedTimout time.Duration
	}{
		{
			name: "trims trailing slashes",
			config: ClientConfig{
				BaseURL: "https://example.com///",
			},
			expectedURL:    "https://example.com",
			expectedTimout: 30 * time.Second,
		},
		{
			name: "custom timeout",
			config: ClientConfig{
				BaseURL: "https://example.com",
				Timeout: 60 * time.Second,
			},
			expectedURL:    "https://example.com",
			expectedTimout: 60 * time.Second,
		},
		{
			name: "default timeout",
			config: ClientConfig{
				BaseURL: "https://example.com",
			},
			expectedURL:    "https://example.com",
			expectedTimout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(tt.config)
			if client.baseURL != tt.expectedURL {
				t.Errorf("Expected baseURL to be '%s', got '%s'", tt.expectedURL, client.baseURL)
			}
			if client.httpClient.Timeout != tt.expectedTimout {
				t.Errorf("Expected timeout to be '%v', got '%v'", tt.expectedTimout, client.httpClient.Timeout)
			}
		})
	}
}

func TestServiceConfigUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ServiceConfig
		wantErr  bool
	}{
		{
			name:     "empty array",
			input:    `[]`,
			expected: ServiceConfig{},
			wantErr:  false,
		},
		{
			name:     "null",
			input:    `null`,
			expected: ServiceConfig{},
			wantErr:  false,
		},
		{
			name:  "object with values",
			input: `{"host": "localhost", "port": 3306}`,
			expected: ServiceConfig{
				"host": "localhost",
				"port": float64(3306),
			},
			wantErr: false,
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: ServiceConfig{},
			wantErr:  false,
		},
		{
			name:     "whitespace with empty array",
			input:    `  []  `,
			expected: ServiceConfig{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sc ServiceConfig
			err := json.Unmarshal([]byte(tt.input), &sc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(sc) != len(tt.expected) {
				t.Errorf("UnmarshalJSON() got %d keys, want %d keys", len(sc), len(tt.expected))
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantErr        bool
		expectedToken  string
	}{
		{
			name:       "successful login",
			statusCode: http.StatusOK,
			responseBody: `{
				"session_token": "test-token-123",
				"id": 1,
				"name": "Admin",
				"is_sys_admin": true
			}`,
			wantErr:       false,
			expectedToken: "test-token-123",
		},
		{
			name:       "invalid credentials",
			statusCode: http.StatusUnauthorized,
			responseBody: `{
				"error": {
					"message": "Invalid credentials"
				}
			}`,
			wantErr:       true,
			expectedToken: "",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			responseBody: `{
				"error": {
					"message": "Internal server error"
				}
			}`,
			wantErr:       true,
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v2/system/admin/session" {
					t.Errorf("Expected path '/api/v2/system/admin/session', got '%s'", r.URL.Path)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			err := client.Login("admin@example.com", "password")

			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client.sessionToken != tt.expectedToken {
				t.Errorf("Expected session token '%s', got '%s'", tt.expectedToken, client.sessionToken)
			}
		})
	}
}

func TestLoginWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to test context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"session_token": "test-token"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.LoginWithContext(ctx, "admin@example.com", "password")
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

func TestGetService(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantErr      bool
		wantNotFound bool
	}{
		{
			name:       "service found",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "test_service",
				"type": "mysql",
				"is_active": true,
				"config": {"host": "localhost", "port": 3306}
			}`,
			wantErr:      false,
			wantNotFound: false,
		},
		{
			name:         "service not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": {"message": "Not found"}}`,
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": {"message": "Server error"}}`,
			wantErr:      true,
			wantNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v2/system/service/1" {
					t.Errorf("Expected path '/api/v2/system/service/1', got '%s'", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			client.sessionToken = "test-token"

			service, err := client.GetService(1)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantNotFound && err != ErrNotFound {
				t.Errorf("Expected ErrNotFound, got %v", err)
			}

			if !tt.wantErr && service == nil {
				t.Error("Expected service to be non-nil")
			}

			if !tt.wantErr && service.Name != "test_service" {
				t.Errorf("Expected service name 'test_service', got '%s'", service.Name)
			}
		})
	}
}

func TestCreateService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/system/service" {
			t.Errorf("Expected path '/api/v2/system/service', got '%s'", r.URL.Path)
		}

		// Verify request body
		var req ServiceCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if len(req.Resource) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(req.Resource))
		}

		if req.Resource[0].Name != "test_service" {
			t.Errorf("Expected service name 'test_service', got '%s'", req.Resource[0].Name)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resource": [{"id": 42}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	service := Service{
		Name:     "test_service",
		Type:     "mysql",
		IsActive: true,
	}

	id, err := client.CreateService(service)
	if err != nil {
		t.Errorf("CreateService() error = %v", err)
		return
	}

	if id != 42 {
		t.Errorf("Expected ID 42, got %d", id)
	}
}

func TestUpdateService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/system/service/1" {
			t.Errorf("Expected path '/api/v2/system/service/1', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	service := Service{
		Name:     "updated_service",
		Type:     "mysql",
		IsActive: true,
	}

	err := client.UpdateService(1, service)
	if err != nil {
		t.Errorf("UpdateService() error = %v", err)
	}
}

func TestDeleteService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/system/service/1" {
			t.Errorf("Expected path '/api/v2/system/service/1', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	err := client.DeleteService(1)
	if err != nil {
		t.Errorf("DeleteService() error = %v", err)
	}
}

func TestGetRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"name": "test_role",
			"description": "Test role",
			"is_active": true
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	role, err := client.GetRole(1)
	if err != nil {
		t.Errorf("GetRole() error = %v", err)
		return
	}

	if role.Name != "test_role" {
		t.Errorf("Expected role name 'test_role', got '%s'", role.Name)
	}
}

func TestCreateRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resource": [{"id": 5}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	role := Role{
		Name:     "new_role",
		IsActive: true,
	}

	id, err := client.CreateRole(role)
	if err != nil {
		t.Errorf("CreateRole() error = %v", err)
		return
	}

	if id != 5 {
		t.Errorf("Expected ID 5, got %d", id)
	}
}

func TestGetApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"name": "test_app",
			"api_key": "abc123",
			"is_active": true,
			"type": 0
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	app, err := client.GetApp(1)
	if err != nil {
		t.Errorf("GetApp() error = %v", err)
		return
	}

	if app.Name != "test_app" {
		t.Errorf("Expected app name 'test_app', got '%s'", app.Name)
	}

	if app.APIKey != "abc123" {
		t.Errorf("Expected API key 'abc123', got '%s'", app.APIKey)
	}
}

func TestCreateApp(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// Create request
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"resource": [{"id": 10}]}`))
		} else {
			// Get request to fetch API key
			if r.Method != "GET" {
				t.Errorf("Expected GET request, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 10,
				"name": "new_app",
				"api_key": "generated-api-key",
				"is_active": true
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.sessionToken = "test-token"

	app := App{
		Name:     "new_app",
		IsActive: true,
	}

	id, apiKey, err := client.CreateApp(app)
	if err != nil {
		t.Errorf("CreateApp() error = %v", err)
		return
	}

	if id != 10 {
		t.Errorf("Expected ID 10, got %d", id)
	}

	if apiKey != "generated-api-key" {
		t.Errorf("Expected API key 'generated-api-key', got '%s'", apiKey)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls (create + get), got %d", callCount)
	}
}

func TestMapStatusToError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		checkErr   func(error) bool
	}{
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       "Resource not found",
			checkErr:   func(err error) bool { return err == ErrNotFound },
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       "Invalid token",
			checkErr:   func(err error) bool { return err != nil },
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       "Invalid input",
			checkErr:   func(err error) bool { return err != nil },
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			body:       "Server crashed",
			checkErr:   func(err error) bool { return err != nil },
		},
		{
			name:       "unknown error",
			statusCode: 418,
			body:       "I'm a teapot",
			checkErr:   func(err error) bool { return err != nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapStatusToError(tt.statusCode, tt.body)
			if !tt.checkErr(err) {
				t.Errorf("mapStatusToError() unexpected error = %v", err)
			}
		})
	}
}
