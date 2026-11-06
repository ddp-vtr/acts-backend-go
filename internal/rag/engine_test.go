package rag

import (
	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProcessQuery_CurrentViewing_ContextLock(t *testing.T) {
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []map[string]string `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var sysMsg string
		for _, m := range payload.Messages {
			if m["role"] == "system" {
				sysMsg = m["content"]
				break
			}
		}

		if !strings.Contains(sysMsg, "[SẢN PHẨM KHÁCH ĐANG XEM TRÊN MÀN HÌNH]") {
			t.Errorf("system prompt missing context lock")
		}
		if !strings.Contains(sysMsg, "BBIA Chaltteok Blush #01") {
			t.Errorf("system prompt missing product name")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"Dạ sản phẩm BBIA Chaltteok Blush #01 Milk Chaltteok có sẵn mẫu thử ạ!"}}]}`))
	}))
	defer mockLLM.Close()

	repo := database.NewRepository()
	engine := NewEngine(repo, mockLLM.URL, "key", "model")

	req := model.ChatRequest{
		SessionID: "sess_101",
		Message:   "có mẫu thử không em?",
		CurrentViewing: &model.ProductView{
			ID:       2,
			Name:     "BBIA Chaltteok Blush #01 Milk Chaltteok",
			Price:    "235.000₫",
			Category: "phan",
		},
	}

	resp, err := engine.ProcessQuery(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.MatchedItems) == 0 {
		t.Fatalf("expected matched items, got 0")
	}

	if resp.MatchedItems[0].ID != 2 || !strings.Contains(resp.MatchedItems[0].Name, "Chaltteok Blush") {
		t.Errorf("expected first matched item to be current viewing product, got: %+v", resp.MatchedItems[0])
	}
}

func TestProcessQuery_SessionTracking_And_CrossSell(t *testing.T) {
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []map[string]string `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&payload)

		var sysMsg string
		for _, m := range payload.Messages {
			if m["role"] == "system" {
				sysMsg = m["content"]
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(sysMsg, "LỊCH SỬ SẢN PHẨM KHÁCH ĐÃ XEM TRONG PHIÊN NÀY") {
			w.Write([]byte(`{"choices":[{"message":{"content":"Dạ em gợi ý Chị dùng thỏi son dưỡng BBIA Lip Oil Chị vừa xem lúc nãy rất hợp với phấn má này ạ!"}}]}`))
		} else {
			w.Write([]byte(`{"choices":[{"message":{"content":"Dạ em chào Chị ạ!"}}]}`))
		}
	}))
	defer mockLLM.Close()

	repo := database.NewRepository()
	engine := NewEngine(repo, mockLLM.URL, "key", "model")
	sessionID := "test_session_cross_sell"

	// Turn 1: Customer views Lip Oil (son)
	_, err := engine.ProcessQuery(model.ChatRequest{
		SessionID: sessionID,
		Message:   "son này có dưỡng nhiều không?",
		CurrentViewing: &model.ProductView{
			ID:       3,
			Name:     "BBIA Lip Oil Essence",
			Price:    "199.000₫",
			Category: "son",
		},
	})
	if err != nil {
		t.Fatalf("turn 1 error: %v", err)
	}

	// Verify session history has product 3
	hist := engine.GetSessionManager().GetSessionHistory(sessionID, 0)
	if len(hist) != 1 || hist[0].ID != 3 {
		t.Fatalf("expected session history to have 1 item (ID 3), got: %+v", hist)
	}

	// Turn 2: Customer navigates to Blush (phan) and asks cross-sell question about son
	resp2, err := engine.ProcessQuery(model.ChatRequest{
		SessionID: sessionID,
		Message:   "chị đang tính mua phấn này, có son nào hợp không em?",
		CurrentViewing: &model.ProductView{
			ID:       2,
			Name:     "BBIA Chaltteok Blush #01 Milk Chaltteok",
			Price:    "235.000₫",
			Category: "phan",
		},
	})
	if err != nil {
		t.Fatalf("turn 2 error: %v", err)
	}

	// Verify MatchedItems index 0 is the previously viewed son (BBIA Lip Oil Essence)
	if len(resp2.MatchedItems) == 0 {
		t.Fatalf("expected matched items in cross-sell response")
	}
	if resp2.MatchedItems[0].ID != 3 || !strings.Contains(resp2.MatchedItems[0].Name, "Lip Oil") {
		t.Errorf("expected cross-sell matched item 0 to be BBIA Lip Oil Essence, got: %+v", resp2.MatchedItems[0])
	}
}

func TestProcessQuery_ConflictingCategoryQuery(t *testing.T) {
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"Dạ các dòng son kem lì hot nhất đây ạ!"}}]}`))
	}))
	defer mockLLM.Close()

	repo := database.NewRepository()
	engine := NewEngine(repo, mockLLM.URL, "key", "model")

	// Viewing Blush (category phan), but explicitly searching for "son lì"
	resp, err := engine.ProcessQuery(model.ChatRequest{
		SessionID: "sess_conflict",
		Message:   "tìm son kem lì giúp chị",
		CurrentViewing: &model.ProductView{
			ID:       2,
			Name:     "BBIA Chaltteok Blush",
			Price:    "235.000₫",
			Category: "phan",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All matched items must be son, blush should NOT be forced to index 0
	for _, item := range resp.MatchedItems {
		if item.Category != "son" {
			t.Errorf("expected all matched items to be 'son', got category %s (%s)", item.Category, item.Name)
		}
	}
}

func TestProductView_Helpers(t *testing.T) {
	pv := &model.ProductView{
		ID:       "2",
		Price:    "235.000₫",
		Category: "phan",
		Variant: map[string]interface{}{
			"code":  "01",
			"shade": "01 Milk Chaltteok",
			"color": "#f8c5c8",
		},
	}

	if pv.GetID() != 2 {
		t.Errorf("expected GetID() = 2, got %d", pv.GetID())
	}
	if pv.GetPriceDisplay() != "235.000₫" {
		t.Errorf("expected GetPriceDisplay() = 235.000₫, got %s", pv.GetPriceDisplay())
	}
	if pv.GetPriceInt() != 235000 {
		t.Errorf("expected GetPriceInt() = 235000, got %d", pv.GetPriceInt())
	}
	if pv.GetVariantCode() != "01" {
		t.Errorf("expected GetVariantCode() = '01', got %s", pv.GetVariantCode())
	}
	if pv.GetVariantColor() != "#f8c5c8" {
		t.Errorf("expected GetVariantColor() = '#f8c5c8', got %s", pv.GetVariantColor())
	}

	pvNum := &model.ProductView{
		ID:    float64(9),
		Price: float64(160000),
	}
	if pvNum.GetID() != 9 {
		t.Errorf("expected GetID() = 9, got %d", pvNum.GetID())
	}
	if pvNum.GetPriceDisplay() != "160.000₫" {
		t.Errorf("expected GetPriceDisplay() = 160.000₫, got %s", pvNum.GetPriceDisplay())
	}
	if pvNum.GetPriceInt() != 160000 {
		t.Errorf("expected GetPriceInt() = 160000, got %d", pvNum.GetPriceInt())
	}
}

