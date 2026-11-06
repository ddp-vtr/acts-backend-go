package model

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

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message     string        `json:"message"`
	ChatHistory []ChatMessage `json:"chat_history,omitempty"`
	Messages    []ChatMessage `json:"messages,omitempty"`
	SessionID   string        `json:"session_id,omitempty"`
	Category    string        `json:"category,omitempty"`
	MaxPrice    int           `json:"max_price,omitempty"`
	Undertone   string        `json:"undertone,omitempty"`
}

type ChatResponse struct {
	Reply        string    `json:"reply"`
	MatchedItems []Product `json:"matched_items"`
	LatencyMs    int64     `json:"latency_ms"`
}

