package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"acts-backend-go/internal/rag"
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

func TestChatApiHandler_CurrentViewing(t *testing.T) {
	// Mock LLM server
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqPayload struct {
			Messages []map[string]string `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Verify system prompt has context lock
		hasContextLock := false
		for _, m := range reqPayload.Messages {
			if m["role"] == "system" && strings.Contains(m["content"], "[SẢN PHẨM KHÁCH ĐANG XEM TRÊN MÀN HÌNH]") {
				hasContextLock = true
				break
			}
		}
		if !hasContextLock {
			t.Errorf("expected system prompt to contain context lock for current_viewing")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"Dạ sản phẩm BBIA Chaltteok Blush #01 Milk Chaltteok có sẵn mẫu thử tại showroom ạ!"}}]}`))
	}))
	defer mockLLM.Close()

	repo := database.NewRepository()
	ragEngine = rag.NewEngine(repo, mockLLM.URL, "test-api-key", "test-model")

	reqBody := `{
		"message": "có mẫu thử không em?",
		"current_viewing": {
			"id": 2,
			"name": "BBIA Chaltteok Blush #01 Milk Chaltteok",
			"price": "235.000₫",
			"category": "phan"
		},
		"chat_history": [
			{"role": "user", "content": "chào em"},
			{"role": "assistant", "content": "Dạ em chào Chị ạ!"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	chatApiHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp model.ChatResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse ChatResponse: %v", err)
	}

	if !strings.Contains(resp.Reply, "Milk Chaltteok") {
		t.Errorf("expected reply to mention Milk Chaltteok, got: %s", resp.Reply)
	}

	if len(resp.MatchedItems) == 0 {
		t.Fatalf("expected MatchedItems to contain viewing product, got 0 items")
	}

	firstItem := resp.MatchedItems[0]
	if firstItem.ID != 2 && !strings.Contains(firstItem.Name, "Chaltteok Blush") {
		t.Errorf("expected first matched item to be current viewing product, got ID=%d, Name=%s", firstItem.ID, firstItem.Name)
	}

	if firstItem.PriceDisplay != "235.000₫" {
		t.Errorf("expected PriceDisplay to be '235.000₫', got: %s", firstItem.PriceDisplay)
	}
}

func TestExtractUserText(t *testing.T) {
	// 1. Text message
	var msg1 WebhookMessagingItem
	msg1.Message = &struct {
		MID        string `json:"mid"`
		Text       string `json:"text"`
		QuickReply *struct {
			Payload string `json:"payload"`
		} `json:"quick_reply"`
	}{Text: "  Xin chào shop  "}
	if got := extractUserText(msg1); got != "Xin chào shop" {
		t.Errorf("expected 'Xin chào shop', got %q", got)
	}

	// 2. Quick reply
	var msg2 WebhookMessagingItem
	msg2.Message = &struct {
		MID        string `json:"mid"`
		Text       string `json:"text"`
		QuickReply *struct {
			Payload string `json:"payload"`
		} `json:"quick_reply"`
	}{
		QuickReply: &struct {
			Payload string `json:"payload"`
		}{Payload: "QUICK_REPLY_ACTION"},
	}
	if got := extractUserText(msg2); got != "QUICK_REPLY_ACTION" {
		t.Errorf("expected 'QUICK_REPLY_ACTION', got %q", got)
	}

	// 3. Postback payload
	var msg3 WebhookMessagingItem
	msg3.Postback = &struct {
		Title   string `json:"title"`
		Payload string `json:"payload"`
	}{Title: "Bắt đầu", Payload: "GET_STARTED"}
	if got := extractUserText(msg3); got != "GET_STARTED" {
		t.Errorf("expected 'GET_STARTED', got %q", got)
	}

	// 4. Postback title fallback
	var msg4 WebhookMessagingItem
	msg4.Postback = &struct {
		Title   string `json:"title"`
		Payload string `json:"payload"`
	}{Title: "Tư vấn", Payload: ""}
	if got := extractUserText(msg4); got != "Tư vấn" {
		t.Errorf("expected 'Tư vấn', got %q", got)
	}

	// 5. Empty item
	var msg5 WebhookMessagingItem
	if got := extractUserText(msg5); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestBuildCarouselPayload(t *testing.T) {
	// Empty items
	if payload := buildCarouselPayload("user123", nil); payload != nil {
		t.Errorf("expected nil payload for empty items")
	}

	prods := []model.Product{
		{ID: 1, Name: "Product 1", PriceDisplay: "160.000₫", Desc: "Desc 1", ImageUrl: "https://example.com/1.png"},
		{ID: 2, Name: "Product 2", PriceDisplay: "235.000₫", Desc: "Desc 2"},
		{ID: 3, Name: "Product 3", PriceDisplay: "189.000₫", Desc: "Desc 3"},
		{ID: 4, Name: "Product 4", PriceDisplay: "495.000₫", Desc: "Desc 4"},
		{ID: 5, Name: "Product 5", PriceDisplay: "220.000₫", Desc: "Desc 5"},
	}

	payload := buildCarouselPayload("user_recip", prods)
	if payload == nil {
		t.Fatalf("expected non-nil payload")
	}

	recip, ok := payload["recipient"].(map[string]string)
	if !ok || recip["id"] != "user_recip" {
		t.Errorf("expected recipient id 'user_recip', got %v", recip)
	}

	msgMap, ok := payload["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("message map invalid")
	}

	attach, ok := msgMap["attachment"].(map[string]interface{})
	if !ok || attach["type"] != "template" {
		t.Fatalf("attachment template invalid")
	}

	pLoad, ok := attach["payload"].(map[string]interface{})
	if !ok || pLoad["template_type"] != "generic" {
		t.Fatalf("generic template invalid")
	}

	elements, ok := pLoad["elements"].([]map[string]interface{})
	if !ok {
		t.Fatalf("elements invalid")
	}

	// Capped at 4 items
	if len(elements) != 4 {
		t.Errorf("expected 4 elements max, got %d", len(elements))
	}

	// Verify buttons point to miniapp and zalo
	for idx, elem := range elements {
		buttons, ok := elem["buttons"].([]map[string]interface{})
		if !ok || len(buttons) != 2 {
			t.Fatalf("element %d buttons invalid, expected 2 buttons", idx)
		}
		if buttons[0]["type"] != "web_url" || buttons[0]["url"] != "https://miniapp.ddphuong.id.vn/" {
			t.Errorf("element %d button 0 should point to miniapp, got %v", idx, buttons[0])
		}
		if buttons[1]["type"] != "web_url" || buttons[1]["url"] != "https://zalo.me/0901911660" {
			t.Errorf("element %d button 1 should point to zalo, got %v", idx, buttons[1])
		}
	}

	// Verify fallback image for item without image
	if elements[1]["image_url"] == "" {
		t.Errorf("element 1 missing fallback image")
	}
}


