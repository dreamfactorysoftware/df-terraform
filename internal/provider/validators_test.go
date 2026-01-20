package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestURLValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid https url",
			value:   "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http url",
			value:   "http://localhost",
			wantErr: false,
		},
		{
			name:    "valid url with port",
			value:   "https://example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid url with path",
			value:   "https://example.com/api",
			wantErr: false,
		},
		{
			name:    "missing scheme",
			value:   "example.com",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			value:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "missing host",
			value:   "https://",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &urlValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %q, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestEmailValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid email",
			value:   "admin@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			value:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			value:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with dots",
			value:   "first.last@example.com",
			wantErr: false,
		},
		{
			name:    "missing @",
			value:   "invalidemail.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			value:   "user@",
			wantErr: true,
		},
		{
			name:    "missing local part",
			value:   "@example.com",
			wantErr: true,
		},
		{
			name:    "missing TLD",
			value:   "user@example",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &emailValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %q, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestPortValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		wantErr bool
	}{
		{
			name:    "valid port 80",
			value:   80,
			wantErr: false,
		},
		{
			name:    "valid port 443",
			value:   443,
			wantErr: false,
		},
		{
			name:    "valid port 3306",
			value:   3306,
			wantErr: false,
		},
		{
			name:    "valid port 65535",
			value:   65535,
			wantErr: false,
		},
		{
			name:    "valid port 1",
			value:   1,
			wantErr: false,
		},
		{
			name:    "invalid port 0",
			value:   0,
			wantErr: true,
		},
		{
			name:    "invalid port negative",
			value:   -1,
			wantErr: true,
		},
		{
			name:    "invalid port too high",
			value:   65536,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &portValidator{}
			req := validator.Int64Request{
				Path:        path.Root("test"),
				ConfigValue: types.Int64Value(tt.value),
			}
			resp := &validator.Int64Response{}

			v.ValidateInt64(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %d, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %d, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestServiceTypeValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "mysql",
			value:   "mysql",
			wantErr: false,
		},
		{
			name:    "pgsql",
			value:   "pgsql",
			wantErr: false,
		},
		{
			name:    "mongodb",
			value:   "mongodb",
			wantErr: false,
		},
		{
			name:    "snowflake",
			value:   "snowflake",
			wantErr: false,
		},
		{
			name:    "MySQL uppercase",
			value:   "MySQL",
			wantErr: false,
		},
		{
			name:    "invalid type",
			value:   "invalid_type",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &serviceTypeValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %q, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestAppTypeValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		wantErr bool
	}{
		{
			name:    "type 0 - no storage",
			value:   0,
			wantErr: false,
		},
		{
			name:    "type 1 - storage service",
			value:   1,
			wantErr: false,
		},
		{
			name:    "type 2 - path",
			value:   2,
			wantErr: false,
		},
		{
			name:    "type 3 - url",
			value:   3,
			wantErr: false,
		},
		{
			name:    "invalid type -1",
			value:   -1,
			wantErr: true,
		},
		{
			name:    "invalid type 4",
			value:   4,
			wantErr: true,
		},
		{
			name:    "invalid type 100",
			value:   100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &appTypeValidator{}
			req := validator.Int64Request{
				Path:        path.Root("test"),
				ConfigValue: types.Int64Value(tt.value),
			}
			resp := &validator.Int64Response{}

			v.ValidateInt64(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %d, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %d, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestIdentifierValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "simple name",
			value:   "myservice",
			wantErr: false,
		},
		{
			name:    "name with underscore",
			value:   "my_service",
			wantErr: false,
		},
		{
			name:    "name with hyphen",
			value:   "my-service",
			wantErr: false,
		},
		{
			name:    "name with numbers",
			value:   "service123",
			wantErr: false,
		},
		{
			name:    "mixed case",
			value:   "MyService",
			wantErr: false,
		},
		{
			name:    "starts with number",
			value:   "123service",
			wantErr: true,
		},
		{
			name:    "starts with underscore",
			value:   "_service",
			wantErr: true,
		},
		{
			name:    "starts with hyphen",
			value:   "-service",
			wantErr: true,
		},
		{
			name:    "contains space",
			value:   "my service",
			wantErr: true,
		},
		{
			name:    "contains special char",
			value:   "my@service",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &identifierValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %q, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

func TestStringNotEmptyValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "normal string",
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "string with spaces",
			value:   "hello world",
			wantErr: false,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "only spaces",
			value:   "   ",
			wantErr: true,
		},
		{
			name:    "only tabs",
			value:   "\t\t",
			wantErr: true,
		},
		{
			name:    "only newlines",
			value:   "\n\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &stringNotEmptyValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error for value %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("Expected no error for value %q, got %v", tt.value, resp.Diagnostics)
			}
		})
	}
}

// Test that validators handle null/unknown values correctly
func TestValidatorsNullUnknown(t *testing.T) {
	ctx := context.Background()

	t.Run("url validator with null", func(t *testing.T) {
		v := &urlValidator{}
		req := validator.StringRequest{
			Path:        path.Root("test"),
			ConfigValue: types.StringNull(),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("Expected no error for null value")
		}
	})

	t.Run("email validator with unknown", func(t *testing.T) {
		v := &emailValidator{}
		req := validator.StringRequest{
			Path:        path.Root("test"),
			ConfigValue: types.StringUnknown(),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("Expected no error for unknown value")
		}
	})

	t.Run("port validator with null", func(t *testing.T) {
		v := &portValidator{}
		req := validator.Int64Request{
			Path:        path.Root("test"),
			ConfigValue: types.Int64Null(),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("Expected no error for null value")
		}
	})

	t.Run("service type validator with unknown", func(t *testing.T) {
		v := &serviceTypeValidator{}
		req := validator.StringRequest{
			Path:        path.Root("test"),
			ConfigValue: types.StringUnknown(),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("Expected no error for unknown value")
		}
	})
}
