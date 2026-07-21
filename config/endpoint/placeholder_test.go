package endpoint

import (
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/gontext"
)

func TestResolvePlaceholder(t *testing.T) {
	result := &Result{
		HTTPStatus:            200,
		IP:                    "127.0.0.1",
		Duration:              250 * time.Millisecond,
		DNSRCode:              "NOERROR",
		Connected:             true,
		CertificateExpiration: 30 * 24 * time.Hour,
		DomainExpiration:      365 * 24 * time.Hour,
		Body:                  []byte(`{"status":"success","items":[1,2,3],"user":{"name":"john","id":123}}`),
		HTTPHeaders: http.Header{
			"Content-Type": []string{"application/json"},
			"Set-Cookie":   []string{"session=abc", "theme=dark"},
		},
	}

	ctx := gontext.New(map[string]interface{}{
		"user_id":       "abc123",
		"session_token": "xyz789",
		"array_data":    []interface{}{"a", "b", "c"},
		"nested": map[string]interface{}{
			"value": "test",
		},
	})

	tests := []struct {
		name        string
		placeholder string
		expected    string
	}{
		// Basic placeholders
		{"status", "[STATUS]", "200"},
		{"ip", "[IP]", "127.0.0.1"},
		{"response-time", "[RESPONSE_TIME]", "250"},
		{"dns-rcode", "[DNS_RCODE]", "NOERROR"},
		{"connected", "[CONNECTED]", "true"},
		{"certificate-expiration", "[CERTIFICATE_EXPIRATION]", "2592000000"},
		{"domain-expiration", "[DOMAIN_EXPIRATION]", "31536000000"},
		{"body", "[BODY]", `{"status":"success","items":[1,2,3],"user":{"name":"john","id":123}}`},

		// Case insensitive placeholders
		{"status-lowercase", "[status]", "200"},
		{"ip-mixed-case", "[Ip]", "127.0.0.1"},

		// Function wrappers on basic placeholders
		{"len-status", "len([STATUS])", "3"},
		{"len-ip", "len([IP])", "9"},
		{"has-status", "has([STATUS])", "true"},
		{"has-empty", "has()", "false"},

		// JSONPath expressions
		{"body-status", "[BODY].status", "success"},
		{"body-user-name", "[BODY].user.name", "john"},
		{"body-user-id", "[BODY].user.id", "123"},
		{"len-body-items", "len([BODY].items)", "3"},
		{"body-array-index", "[BODY].items[0]", "1"},
		{"has-body-status", "has([BODY].status)", "true"},
		{"has-body-missing", "has([BODY].missing)", "false"},

		// Header placeholders
		{"header-single-value", "[HEADERS].Content-Type", "application/json"},
		{"header-case-insensitive-name", "[HEADERS].content-type", "application/json"},
		{"header-repeated-values", "[HEADERS].Set-Cookie", `["session=abc","theme=dark"]`},
		{"header-missing", "[HEADERS].X-Missing", "[HEADERS].X-Missing (INVALID)"},
		{"len-header", "len([HEADERS].Content-Type)", "16"},
		{"len-header-repeated", "len([HEADERS].Set-Cookie)", "2"},
		{"len-header-missing", "len([HEADERS].X-Missing)", "0"},
		{"has-header-present", "has([HEADERS].Content-Type)", "true"},
		{"has-header-missing", "has([HEADERS].X-Missing)", "false"},

		// Bare [HEADERS] placeholder (every header in the response)
		{"headers-bare-has", "has([HEADERS])", "true"},
		{"headers-bare-len-distinct-names", "len([HEADERS])", "2"}, // Content-Type, Set-Cookie
		{"headers-bare-resolve", "[HEADERS]", `{"Content-Type":"application/json","Set-Cookie":["session=abc","theme=dark"]}`},

		// Context placeholders
		{"context-user-id", "[CONTEXT].user_id", "abc123"},
		{"context-session-token", "[CONTEXT].session_token", "xyz789"},
		{"context-nested", "[CONTEXT].nested.value", "test"},
		{"len-context-array", "len([CONTEXT].array_data)", "3"},
		{"has-context-user-id", "has([CONTEXT].user_id)", "true"},
		{"has-context-missing", "has([CONTEXT].missing)", "false"},

		// Invalid placeholders
		{"unknown-placeholder", "[UNKNOWN]", "[UNKNOWN]"},
		{"len-unknown", "len([UNKNOWN])", "len([UNKNOWN]) (INVALID)"},
		{"has-unknown", "has([UNKNOWN])", "false"},
		{"invalid-jsonpath", "[BODY].invalid.path", "[BODY].invalid.path (INVALID)"},

		// Literal strings
		{"literal-string", "literal", "literal"},
		{"number-string", "123", "123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ResolvePlaceholder(test.placeholder, result, ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if actual != test.expected {
				t.Errorf("expected '%s', got '%s'", test.expected, actual)
			}
		})
	}
}

func TestResolvePlaceholderWithoutContext(t *testing.T) {
	result := &Result{
		HTTPStatus: 404,
		Body:       []byte(`{"error":"not found"}`),
	}

	tests := []struct {
		name        string
		placeholder string
		expected    string
	}{
		{"status-without-context", "[STATUS]", "404"},
		{"body-without-context", "[BODY].error", "not found"},
		{"context-without-context", "[CONTEXT].user_id", "[CONTEXT].user_id"},
		{"has-context-without-context", "has([CONTEXT].user_id)", "false"},
		{"header-without-headers", "[HEADERS].Content-Type", "[HEADERS].Content-Type (INVALID)"},
		{"has-header-without-headers", "has([HEADERS].Content-Type)", "false"},
		{"headers-bare-without-headers", "[HEADERS]", "[HEADERS] (INVALID)"},
		{"headers-bare-has-without-headers", "has([HEADERS])", "false"},
		{"headers-bare-len-without-headers", "len([HEADERS])", "0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ResolvePlaceholder(test.placeholder, result, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if actual != test.expected {
				t.Errorf("expected '%s', got '%s'", test.expected, actual)
			}
		})
	}
}
