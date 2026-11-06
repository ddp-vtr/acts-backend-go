package rag

import (
	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Engine struct {
	repo        *database.Repository
	llmEndpoint string
	llmApiKey   string
	llmModel    string
	httpClient  *http.Client
}

func NewEngine(repo *database.Repository, endpoint, apiKey, modelName string) *Engine {
	return &Engine{
		repo:        repo,
		llmEndpoint: endpoint,
		llmApiKey:   apiKey,
		llmModel:    modelName,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// 1. Trích xuất Intent & Slot nhanh (Fast Slot Extraction)
func (e *Engine) extractSlots(userMsg string) (category string, maxPrice int, undertone string) {
	lower := strings.ToLower(userMsg)

	// Category
	if strings.Contains(lower, "son") || strings.Contains(lower, "tint") || strings.Contains(lower, "môi") {
		category = "son"
	} else if strings.Contains(lower, "phấn") || strings.Contains(lower, "kiềm dầu") || strings.Contains(lower, "má hồng") {
		category = "phan"
	} else if strings.Contains(lower, "kẻ mắt") || strings.Contains(lower, "eyeliner") || strings.Contains(lower, "mascara") || strings.Contains(lower, "mắt") {
		category = "mat"
	} else if strings.Contains(lower, "cushion") {
		category = "phan"
	}

	// Undertone
	if strings.Contains(lower, "ấm") || strings.Contains(lower, "warm") {
		undertone = "warm"
	} else if strings.Contains(lower, "lạnh") || strings.Contains(lower, "cool") {
		undertone = "cool"
	}

	// Price regex (ví dụ: "dưới 200k", "<200000", "200.000")
	rePrice := regexp.MustCompile(`(?:dưới|<|khoảng)?\s*(\d+)\s*(?:k|nghìn|000)`)
	matches := rePrice.FindStringSubmatch(lower)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			if val < 1000 {
				maxPrice = val * 1000
			} else {
				maxPrice = val
			}
		}
	}

	return
}

// 2. Thực hiện Hybrid Search + Smart Fallback cho câu hỏi mở
func (e *Engine) ProcessQuery(req model.ChatRequest) (*model.ChatResponse, error) {
	start := time.Now()

	userMsg := strings.TrimSpace(req.Message)
	var history []model.ChatMessage

	if len(req.ChatHistory) > 0 {
		history = req.ChatHistory
	} else if len(req.Messages) > 0 {
		for i, m := range req.Messages {
			if m.Role == "system" {
				continue
			}
			if i == len(req.Messages)-1 && userMsg == "" {
				userMsg = strings.TrimSpace(m.Content)
			} else {
				history = append(history, m)
			}
		}
	}

	cat, maxP, un := e.extractSlots(userMsg)
	if req.Category != "" {
		cat = req.Category
	}
	if req.MaxPrice > 0 {
		maxP = req.MaxPrice
	}
	if req.Undertone != "" {
		un = req.Undertone
	}

	// Lấy ngữ cảnh danh mục từ các lượt chat trước nếu lượt này khách hỏi tiếp (ví dụ: "màu 25", "cho xem thêm")
	if cat == "" {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == "user" {
				prevCat, _, _ := e.extractSlots(history[i].Content)
				if prevCat != "" {
					cat = prevCat
					break
				}
			}
		}
	}

	lowerMsg := strings.ToLower(userMsg)
	isPolicyOrHours := strings.Contains(lowerMsg, "mở cửa") || strings.Contains(lowerMsg, "đóng cửa") ||
		strings.Contains(lowerMsg, "giờ") || strings.Contains(lowerMsg, "đổi trả") ||
		strings.Contains(lowerMsg, "bảo hành") || strings.Contains(lowerMsg, "hoàn tiền") ||
		strings.Contains(lowerMsg, "trả hàng") || strings.Contains(lowerMsg, "lỗi") ||
		strings.Contains(lowerMsg, "gãy") || strings.Contains(lowerMsg, "vỡ")

	// Tìm kiếm sản phẩm phù hợp
	matchedItems := e.repo.SearchProducts(cat, maxP, un, userMsg, 3)

	var catalogContext strings.Builder
	if len(matchedItems) > 0 {
		catalogContext.WriteString("\n[DANH MỤC SẢN PHẨM PHÙ HỢP CÒN HÀNG TRONG KHO]:\n")
		for _, item := range matchedItems {
			catalogContext.WriteString(fmt.Sprintf("- Tên: %s | Giá: %s | Tồn kho: %d | Chi tiết: %s\n",
				item.Name, item.PriceDisplay, item.StockQuantity, item.Desc))
		}
	} else if !isPolicyOrHours {
		// SMART FALLBACK: Khách hỏi chung ("có bao nhiêu sp", "tư vấn giúp", "shop có gì")
		matchedItems = e.repo.GetFeaturedProducts(3)
		catalogContext.WriteString("\n[LƯU Ý]: Khách đang hỏi câu hỏi chung hoặc chào hỏi. Hãy tự nhiên giới thiệu các dòng sản phẩm K-Beauty hot nhất hiện có:\n")
		for _, item := range matchedItems {
			catalogContext.WriteString(fmt.Sprintf("- Tên: %s | Giá: %s | Chi tiết: %s\n",
				item.Name, item.PriceDisplay, item.Desc))
		}
	}

	// System prompt đồng bộ Persona chuẩn K-Beauty & dữ liệu chuẩn sky007.vn
	greetingRule := "1. Xưng 'em', gọi khách là 'Chị' (hoặc 'bạn đẹp'), giọng điệu ngọt ngào, tinh tế, lịch thiệp."
	if len(history) > 0 {
		greetingRule += "\n   - ĐÃ CÓ LỊCH SỬ HỘI THOẠI TRƯỚC ĐÓ: TUYỆT ĐỐI KHÔNG lặp lại câu chào hỏi ('Dạ em chào Chị ạ'). Hãy trả lời thẳng thắn, tiếp nối tự nhiên với ngữ cảnh trước."
	} else {
		greetingRule += "\n   - LƯỢT ĐẦU TIÊN: Chào nhẹ 'Dạ em chào Chị ạ'."
	}

	systemPrompt := `Bạn là chuyên viên tư vấn sắc đẹp K-Beauty của SKY / sky007.vn (phân phối độc quyền BBIA, About Tone tại Việt Nam bởi ActsOne Việt Nam).
QUY TẮC BẮT BUỘC:
` + greetingRule + `
2. THÔNG TIN CHUẨN XÁC TỪ SKY007.VN:
   - Đơn vị: Công ty TNHH ActsOne Việt Nam (Nhà phân phối độc quyền BBIA tại VN từ 2011).
   - Địa chỉ: Số 20, đường 01, KĐT mới Him Lam, P. Tân Hưng, TP.HCM.
   - Giờ làm việc văn phòng & kho: 09:00 – 18:00 (Thứ 2 – Thứ 6). Thứ 7, CN nghỉ hành chính.
   - Hotline & Zalo CSKH: 028 6274 0330 · 0901 911 660.
   - Mua sắm Online / Bot AI: Trực 24/7 nhận đơn (đặt đêm ngày mai kho ưu tiên đóng gói giao sớm).
   - Chính sách đổi trả & hoàn tiền: Trong vòng 7 ngày kể từ khi nhận hàng. Đổi mới 100% và sky007 chịu phí ship 2 đầu nếu lỗi do NSX, vỡ/gãy son hoặc giao sai màu (yêu cầu còn nguyên đai kiện, có video clip mở hộp unbox). Khách muốn đổi tone màu khác chịu phí ship. Có hỗ trợ hoàn tiền 100% qua STK ngân hàng.
3. Tuyệt đối KHÔNG bao giờ nói 'kho hết hàng' khi khách hỏi câu chung chung. Hãy nhiệt tình giới thiệu các sản phẩm hot bên dưới.
4. Chỉ tư vấn sản phẩm có trong danh mục cung cấp bên dưới. Không bịa giá, không dùng từ cấm y tế.
5. Trả lời súc tích, dưới 90 chữ.` + catalogContext.String()

	llmReply, err := e.callLLM(systemPrompt, userMsg, history)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start).Milliseconds()

	// Nếu khách chỉ hỏi giờ mở cửa hoặc chính sách đổi trả -> không đính kèm card carousel
	if isPolicyOrHours && cat == "" {
		matchedItems = []model.Product{}
	}
	if matchedItems == nil {
		matchedItems = []model.Product{}
	}

	return &model.ChatResponse{
		Reply:        llmReply,
		MatchedItems: matchedItems,
		LatencyMs:    elapsed,
	}, nil
}

func (e *Engine) callLLM(systemPrompt, userText string, history []model.ChatMessage) (string, error) {
	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}

	// Lấy tối đa 8 lượt trò chuyện gần nhất để tối ưu token
	histLen := len(history)
	startIdx := 0
	if histLen > 8 {
		startIdx = histLen - 8
	}
	for _, h := range history[startIdx:] {
		role := h.Role
		if role == "" {
			role = "user"
		}
		if h.Content != "" {
			messages = append(messages, map[string]string{
				"role":    role,
				"content": h.Content,
			})
		}
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userText,
	})

	payload := map[string]interface{}{
		"model":      e.llmModel,
		"stream":     false,
		"max_tokens": 300,
		"messages":   messages,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", e.llmEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if e.llmApiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.llmApiKey)
	}

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("JSON parse error: %v (raw: %s)", err, string(respBody))
	}

	if parsed.Error.Message != "" {
		return "", fmt.Errorf("LLM error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) > 0 {
		return parsed.Choices[0].Message.Content, nil
	}

	return "Dạ em chào Chị ạ! Em là tư vấn viên của SKY, Chị đang quan tâm son kem lì, phấn má hay kẻ mắt để em hỗ trợ mình ngay nhé!", nil
}
