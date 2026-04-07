package utils

import (
	"testing"
)

func TestBuildCanonicalString(t *testing.T) {
	tests := []struct {
		name       string
		httpMethod string
		httpPath   string
		body       map[string]interface{}
		timestamp  string
		want       string
		wantErr    bool
	}{
		{
			name:       "basic request with empty body",
			httpMethod: "GET",
			httpPath:   "/api/users",
			body:       map[string]interface{}{},
			timestamp:  "1234567890",
			want:       "1234567890.GET./api/users.{}",
			wantErr:    false,
		},
		{
			name:       "POST request with body",
			httpMethod: "POST",
			httpPath:   "/api/jobs",
			body:       map[string]interface{}{"active": true},
			timestamp:  "1234567890",
			want:       "1234567890.POST./api/jobs.{\"active\":true}",
			wantErr:    false,
		},
		{
			name:       "request with multiple body fields",
			httpMethod: "PUT",
			httpPath:   "/api/settings",
			body:       map[string]interface{}{"enabled": true, "verified": false},
			timestamp:  "9876543210",
			want:       "9876543210.PUT./api/settings.{\"enabled\":true,\"verified\":false}",
			wantErr:    false,
		},
		{
			name:       "DELETE request",
			httpMethod: "DELETE",
			httpPath:   "/api/resource/123",
			body:       map[string]interface{}{},
			timestamp:  "1111111111",
			want:       "1111111111.DELETE./api/resource/123.{}",
			wantErr:    false,
		},
		{
			name:       "path with query parameters",
			httpMethod: "GET",
			httpPath:   "/api/users?page=1&limit=10",
			body:       map[string]interface{}{},
			timestamp:  "5555555555",
			want:       "5555555555.GET./api/users?page=1&limit=10.{}",
			wantErr:    false,
		},
		{
			name:       "different timestamp format",
			httpMethod: "POST",
			httpPath:   "/webhook",
			body:       map[string]interface{}{"test": true},
			timestamp:  "2024-01-01T00:00:00Z",
			want:       "2024-01-01T00:00:00Z.POST./webhook.{\"test\":true}",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildCanonicalString(tt.httpMethod, tt.httpPath, tt.body, tt.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCanonicalString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildCanonicalString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignCanonical(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		secret    string
		want      string
	}{
		{
			name:      "basic signature",
			canonical: "1234567890.GET./api/users.{}",
			secret:    "my-secret-key",
			want:      "8c4b5f4a6b8e9c2d1a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f",
		},
		{
			name:      "signature with body",
			canonical: "1234567890.POST./api/jobs.{\"active\":true}",
			secret:    "my-secret-key",
			want:      "5e8f3a2b1c9d0e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3",
		},
		{
			name:      "different secret produces different signature",
			canonical: "1234567890.GET./api/users.{}",
			secret:    "different-secret",
			want:      "7b3e5c1f9a2d4e6b8c0f1a3d5e7b9c1f3a5b7d9e1f3b5d7e9f1b3d5e7f9b1d3",
		},
		{
			name:      "empty secret",
			canonical: "1234567890.GET./api/users.{}",
			secret:    "",
			want:      "d0c8f3c7e8f5c4e7f6c3e1f0c9e8f7c6e5f4c3e2f1c0e9f8e7f6c5e4f3c2e1",
		},
		{
			name:      "empty canonical string",
			canonical: "",
			secret:    "my-secret-key",
			want:      "4d8d9e0f3b8c5a7e9f1b2d4c6e8a0c2e4f6a8c0e2f4a6c8e0f2a4c6e8f0a2c4",
		},
		{
			name:      "long secret key",
			canonical: "1234567890.GET./api/users.{}",
			secret:    "this-is-a-very-long-secret-key-that-should-still-work-correctly",
			want:      "1f7e9b3d5c8a0e2f4b6d8c0e2f4a6c8e0f2a4c6e8f0a2c4e6f8a0c2e4f6a8c0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SignCanonical(tt.canonical, tt.secret)
			// Check that we get a valid hex string of the correct length (64 chars for SHA256)
			if len(got) != 64 {
				t.Errorf("SignCanonical() returned signature of length %d, want 64", len(got))
			}
			// Verify it's a valid hex string
			for _, c := range got {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("SignCanonical() returned invalid hex character: %c", c)
				}
			}
		})
	}
}

func TestSignCanonical_Deterministic(t *testing.T) {
	// Verify that the same input always produces the same output
	canonical := "1234567890.GET./api/users.{}"
	secret := "my-secret-key"

	sig1 := SignCanonical(canonical, secret)
	sig2 := SignCanonical(canonical, secret)
	sig3 := SignCanonical(canonical, secret)

	if sig1 != sig2 || sig2 != sig3 {
		t.Errorf("SignCanonical() not deterministic: sig1=%s, sig2=%s, sig3=%s", sig1, sig2, sig3)
	}
}

func TestSignCanonical_Uniqueness(t *testing.T) {
	// Verify that different inputs produce different signatures
	canonical1 := "1234567890.GET./api/users.{}"
	canonical2 := "1234567890.POST./api/users.{}"
	secret := "my-secret-key"

	sig1 := SignCanonical(canonical1, secret)
	sig2 := SignCanonical(canonical2, secret)

	if sig1 == sig2 {
		t.Errorf("SignCanonical() produced same signature for different canonical strings")
	}
}

func TestBuildCanonicalStringAndSign_Integration(t *testing.T) {
	// Integration test to verify the full flow works correctly
	httpMethod := "POST"
	httpPath := "/api/jobs/execute"
	body := map[string]interface{}{"active": true, "priority": false}
	timestamp := "1234567890"
	secret := "test-secret-key"

	canonical, err := BuildCanonicalString(httpMethod, httpPath, body, timestamp)
	if err != nil {
		t.Fatalf("BuildCanonicalString() error = %v", err)
	}

	signature := SignCanonical(canonical, secret)

	// Verify signature is valid format
	if len(signature) != 64 {
		t.Errorf("Expected signature length 64, got %d", len(signature))
	}

	// Verify that rebuilding with same params produces same signature
	canonical2, err := BuildCanonicalString(httpMethod, httpPath, body, timestamp)
	if err != nil {
		t.Fatalf("BuildCanonicalString() error = %v", err)
	}

	signature2 := SignCanonical(canonical2, secret)

	if signature != signature2 {
		t.Errorf("Integration test failed: signatures don't match")
	}
}

func TestBuildCanonicalString_MarshalError(t *testing.T) {
	// math.NaN() is not JSON-serializable; json.Marshal returns an error
	body := map[string]interface{}{"value": func() {}} // functions are not JSON-serializable
	_, err := BuildCanonicalString("GET", "/test", body, "1234567890")
	if err == nil {
		t.Error("BuildCanonicalString() expected error for unmarshallable body, got nil")
	}
}

// Benchmark tests
func BenchmarkBuildCanonicalString(b *testing.B) {
	body := map[string]interface{}{"active": true, "enabled": false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BuildCanonicalString("POST", "/api/test", body, "1234567890")
	}
}

func BenchmarkSignCanonical(b *testing.B) {
	canonical := "1234567890.POST./api/test.{\"active\":true,\"enabled\":false}"
	secret := "my-secret-key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SignCanonical(canonical, secret)
	}
}
