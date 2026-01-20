package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// urlValidator validates that a string is a valid URL
type urlValidator struct{}

func (v *urlValidator) Description(ctx context.Context) string {
	return "value must be a valid URL (e.g., https://example.com)"
}

func (v *urlValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid URL (e.g., `https://example.com`)"
}

func (v *urlValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	parsedURL, err := url.Parse(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("The value %q is not a valid URL: %s", value, err),
		)
		return
	}

	if parsedURL.Scheme == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("The URL %q must include a scheme (http:// or https://)", value),
		)
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL Scheme",
			fmt.Sprintf("The URL scheme must be 'http' or 'https', got %q", parsedURL.Scheme),
		)
		return
	}

	if parsedURL.Host == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("The URL %q must include a host", value),
		)
		return
	}
}

// emailValidator validates that a string is a valid email address
type emailValidator struct{}

func (v *emailValidator) Description(ctx context.Context) string {
	return "value must be a valid email address"
}

func (v *emailValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid email address"
}

func (v *emailValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Basic email validation - must contain @ and have text on both sides
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Email Address",
			fmt.Sprintf("The value %q is not a valid email address", value),
		)
	}
}

// portValidator validates that an integer is a valid port number
type portValidator struct{}

func (v *portValidator) Description(ctx context.Context) string {
	return "value must be a valid port number (1-65535)"
}

func (v *portValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid port number (1-65535)"
}

func (v *portValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueInt64()

	if value < 1 || value > 65535 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Port Number",
			fmt.Sprintf("Port must be between 1 and 65535, got %d", value),
		)
	}
}

// serviceTypeValidator validates that a string is a valid DreamFactory service type
type serviceTypeValidator struct{}

// Valid database service types in DreamFactory
var validServiceTypes = map[string]bool{
	// SQL databases
	"mysql":      true,
	"pgsql":      true,
	"sqlite":     true,
	"sqlsrv":     true,
	"oracle":     true,
	"ibmdb2":     true,
	"informix":   true,
	"firebird":   true,
	"sqlanywhere": true,

	// NoSQL databases
	"mongodb":     true,
	"couchdb":     true,
	"dynamodb":    true,
	"cassandra":   true,

	// Cloud databases
	"aws_redshift":     true,
	"aws_dynamodb":     true,
	"azure_documentdb": true,
	"azure_table":      true,

	// Other services
	"email":   true,
	"ldap":    true,
	"adldap":  true,
	"salesforce": true,
	"cache":   true,
	"logstash": true,

	// Snowflake
	"snowflake": true,
}

func (v *serviceTypeValidator) Description(ctx context.Context) string {
	return "value must be a valid DreamFactory service type"
}

func (v *serviceTypeValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid DreamFactory service type (e.g., `mysql`, `pgsql`, `mongodb`, `snowflake`)"
}

func (v *serviceTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	lowerValue := strings.ToLower(value)

	if !validServiceTypes[lowerValue] {
		// Build a list of valid types for the error message
		validTypes := make([]string, 0, len(validServiceTypes))
		for t := range validServiceTypes {
			validTypes = append(validTypes, t)
		}

		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Service Type",
			fmt.Sprintf("The service type %q is not recognized. Valid types include: mysql, pgsql, mongodb, snowflake, sqlsrv, oracle, etc.", value),
		)
	}
}

// appTypeValidator validates that an integer is a valid DreamFactory app type
type appTypeValidator struct{}

func (v *appTypeValidator) Description(ctx context.Context) string {
	return "value must be a valid app type (0-3)"
}

func (v *appTypeValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid app type: 0=No storage, 1=Storage service, 2=Path, 3=URL"
}

func (v *appTypeValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueInt64()

	if value < 0 || value > 3 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid App Type",
			fmt.Sprintf("App type must be 0 (No storage), 1 (Storage service), 2 (Path), or 3 (URL), got %d", value),
		)
	}
}

// stringNotEmptyValidator validates that a string is not empty
type stringNotEmptyValidator struct{}

func (v *stringNotEmptyValidator) Description(ctx context.Context) string {
	return "value must not be empty"
}

func (v *stringNotEmptyValidator) MarkdownDescription(ctx context.Context) string {
	return "value must not be empty"
}

func (v *stringNotEmptyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	if strings.TrimSpace(value) == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Value Cannot Be Empty",
			"This field cannot be empty or contain only whitespace",
		)
	}
}

// identifierValidator validates that a string is a valid identifier (for names)
type identifierValidator struct{}

func (v *identifierValidator) Description(ctx context.Context) string {
	return "value must be a valid identifier (letters, numbers, underscores, hyphens)"
}

func (v *identifierValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid identifier (letters, numbers, underscores, hyphens)"
}

func (v *identifierValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Check for empty string
	if value == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Identifier",
			"Identifier cannot be empty",
		)
		return
	}

	// Must start with a letter
	if !regexp.MustCompile(`^[a-zA-Z]`).MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Identifier",
			fmt.Sprintf("Identifier %q must start with a letter", value),
		)
		return
	}

	// Can only contain letters, numbers, underscores, and hyphens
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]*$`).MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Identifier",
			fmt.Sprintf("Identifier %q can only contain letters, numbers, underscores, and hyphens", value),
		)
	}
}
