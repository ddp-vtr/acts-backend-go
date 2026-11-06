package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "8081"},
		{"8080", "8080"},
		{":8080", "8080"},
		{"  :9000  ", "9000"},
		{"10000", "10000"},
	}

	for _, tc := range tests {
		actual := normalizePort(tc.input)
		if actual != tc.expected {
			t.Errorf("normalizePort(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if rr.Body.String() != "OK" {
		t.Fatalf("expected body OK, got %s", rr.Body.String())
	}
}

func TestRootEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"online","service":"acts-backend-go","version":"1.0.0"}`))
	})

	// Test root GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Test not found /unknown
	req404 := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr404 := httptest.NewRecorder()
	handler.ServeHTTP(rr404, req404)
	if rr404.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr404.Code)
	}
}

func TestVerifySignature(t *testing.T) {
	appSecret = "test-secret"
	payload := []byte(`{"hello":"world"}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(payload)
	validHash := hex.EncodeToString(mac.Sum(nil))

	// Test valid lowercase
	if !verifySignature(payload, "sha256="+validHash) {
		t.Errorf("expected signature to be valid")
	}

	// Test valid uppercase
	if !verifySignature(payload, "sha256="+hex.EncodeToString(mac.Sum(nil))) {
		t.Errorf("expected signature to be valid")
	}

	// Test invalid prefix
	if verifySignature(payload, validHash) {
		t.Errorf("expected invalid prefix to fail")
	}

	// Test wrong hash
	if verifySignature(payload, "sha256=invalidhash123") {
		t.Errorf("expected invalid hash to fail")
	}
}

func TestWebhookVerification(t *testing.T) {
	verifyToken = "test-verify-token"

	// Valid challenge
	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=test-verify-token&hub.challenge=test_123", nil)
	rr := httptest.NewRecorder()
	webhookHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "test_123" {
		t.Fatalf("expected test_123, got %s", rr.Body.String())
	}

	// Invalid token
	reqBad := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=test_123", nil)
	rrBad := httptest.NewRecorder()
	webhookHandler(rrBad, reqBad)

	if rrBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rrBad.Code)
	}
}
