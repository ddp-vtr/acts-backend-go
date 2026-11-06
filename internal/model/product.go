package model

import (
	"fmt"
	"strconv"
	"strings"
)

type Product struct {
	ID            int      `json:"id"`
	SkyID         int      `json:"sky_id"`
	Category      string   `json:"category"`
	Brand         string   `json:"brand"`
	Name          string   `json:"name"`
	Desc          string   `json:"desc"`
	Price         int      `json:"price"` // Giá số nguyên để lọc SQL chuẩn xác (<, >, BETWEEN)
	PriceDisplay  string   `json:"price_display"`
	StockQuantity int      `json:"stock_quantity"` // Tồn kho thực tế
	ImageUrl      string   `json:"image_url"`
	Undertone     string   `json:"undertone"` // warm / cool / neutral
	Keywords      []string `json:"keywords"`
	Code          string   `json:"code,omitempty"`
	Hex           string   `json:"hex,omitempty"`
}

type ProductView struct {
	ID       any    `json:"id"`
	Name     string `json:"name"`
	Price    any    `json:"price"`
	Category string `json:"category,omitempty"`
	Img      string `json:"img,omitempty"`
	Desc     string `json:"desc,omitempty"`
	Variant  any    `json:"variant,omitempty"`
}

func (pv *ProductView) GetID() int {
	if pv == nil || pv.ID == nil {
		return 0
	}
	switch v := pv.ID.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	}
	return 0
}

func (pv *ProductView) GetPriceDisplay() string {
	if pv == nil || pv.Price == nil {
		return ""
	}
	switch v := pv.Price.(type) {
	case string:
		s := strings.TrimSpace(v)
		if strings.HasSuffix(s, "₫") || strings.HasSuffix(s, "đ") {
			return s
		}
		cleaned := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, s)
		if n, err := strconv.Atoi(cleaned); err == nil && n > 0 {
			if n < 1000 {
				return fmt.Sprintf("%d.000₫", n)
			}
			return fmt.Sprintf("%d.000₫", n/1000)
		}
		return s
	case float64:
		val := int(v)
		if val < 1000 && val > 0 {
			return fmt.Sprintf("%d.000₫", val)
		}
		return fmt.Sprintf("%d.000₫", val/1000)
	case int:
		val := v
		if val < 1000 && val > 0 {
			return fmt.Sprintf("%d.000₫", val)
		}
		return fmt.Sprintf("%d.000₫", val/1000)
	}
	return fmt.Sprintf("%v", pv.Price)
}

func (pv *ProductView) GetPriceInt() int {
	if pv == nil || pv.Price == nil {
		return 0
	}
	switch v := pv.Price.(type) {
	case int:
		if v < 1000 && v > 0 {
			return v * 1000
		}
		return v
	case float64:
		val := int(v)
		if val < 1000 && val > 0 {
			return val * 1000
		}
		return val
	case string:
		cleaned := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, v)
		if n, err := strconv.Atoi(cleaned); err == nil {
			if n < 1000 && n > 0 {
				return n * 1000
			}
			return n
		}
	}
	return 0
}

func (pv *ProductView) GetVariantCode() string {
	if pv == nil || pv.Variant == nil {
		return ""
	}
	switch v := pv.Variant.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		if c, ok := v["code"].(string); ok {
			return strings.TrimSpace(c)
		}
	case map[string]string:
		if c, ok := v["code"]; ok {
			return strings.TrimSpace(c)
		}
	}
	return ""
}

func (pv *ProductView) GetVariantColor() string {
	if pv == nil || pv.Variant == nil {
		return ""
	}
	switch v := pv.Variant.(type) {
	case map[string]interface{}:
		if hex, ok := v["color"].(string); ok && hex != "" {
			return hex
		}
		if hex, ok := v["hex"].(string); ok && hex != "" {
			return hex
		}
	case map[string]string:
		if hex, ok := v["color"]; ok && hex != "" {
			return hex
		}
		if hex, ok := v["hex"]; ok && hex != "" {
			return hex
		}
	}
	return ""
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message        string              `json:"message"`
	CurrentViewing *ProductView        `json:"current_viewing,omitempty"`
	ChatHistory    []map[string]string `json:"chat_history,omitempty"`
	Messages       []ChatMessage       `json:"messages,omitempty"`
	SessionID      string              `json:"session_id,omitempty"`
	Category       string              `json:"category,omitempty"`
	MaxPrice       int                 `json:"max_price,omitempty"`
	Undertone      string              `json:"undertone,omitempty"`
}

type ChatResponse struct {
	Reply        string    `json:"reply"`
	MatchedItems []Product `json:"matched_items"`
	LatencyMs    int64     `json:"latency_ms"`
}
