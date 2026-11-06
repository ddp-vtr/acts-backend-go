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
	"sync"
	"time"
)

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string][]model.Product
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string][]model.Product),
	}
}

func (sm *SessionManager) RecordProductView(sessionID string, prod model.Product) {
	if sessionID == "" {
		sessionID = "default"
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()

	history := sm.sessions[sessionID]
	var filtered []model.Product
	for _, p := range history {
		if p.ID != prod.ID && !strings.EqualFold(p.Name, prod.Name) {
			filtered = append(filtered, p)
		}
	}
	filtered = append(filtered, prod)
	if len(filtered) > 10 {
		filtered = filtered[len(filtered)-10:]
	}
	sm.sessions[sessionID] = filtered
}

func (sm *SessionManager) GetSessionHistory(sessionID string, excludeID int) []model.Product {
	if sessionID == "" {
		sessionID = "default"
	}
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	history := sm.sessions[sessionID]
	var res []model.Product
	for _, p := range history {
		if excludeID <= 0 || p.ID != excludeID {
			res = append(res, p)
		}
	}
	return res
}

type Engine struct {
	repo        *database.Repository
	sessionMgr  *SessionManager
	llmEndpoint string
	llmApiKey   string
	llmModel    string
	httpClient  *http.Client
}

func NewEngine(repo *database.Repository, endpoint, apiKey, modelName string) *Engine {
	return &Engine{
		repo:        repo,
		sessionMgr:  NewSessionManager(),
		llmEndpoint: endpoint,
		llmApiKey:   apiKey,
		llmModel:    modelName,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (e *Engine) GetSessionManager() *SessionManager {
	return e.sessionMgr
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
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "default"
	}

	var history []model.ChatMessage

	if len(req.ChatHistory) > 0 {
		for _, m := range req.ChatHistory {
			role := m["role"]
			if role == "" {
				role = "user"
			}
			content := m["content"]
			if content != "" {
				history = append(history, model.ChatMessage{
					Role:    role,
					Content: content,
				})
			}
		}
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

	// 1. Nhận diện sản phẩm khách đang xem (current_viewing) nếu có
	var viewProd *model.Product
	if req.CurrentViewing != nil && (req.CurrentViewing.Name != "" || req.CurrentViewing.GetID() > 0) {
		viewID := req.CurrentViewing.GetID()
		if viewID > 0 {
			if p := e.repo.GetProductByID(viewID); p != nil {
				vp := *p
				viewProd = &vp
			}
		}
		if viewProd == nil && req.CurrentViewing.Name != "" {
			if p := e.repo.FindProductByName(req.CurrentViewing.Name); p != nil {
				vp := *p
				viewProd = &vp
			}
		}
		if viewProd == nil {
			viewProd = &model.Product{
				ID:            viewID,
				Name:          req.CurrentViewing.Name,
				Category:      req.CurrentViewing.Category,
				Desc:          req.CurrentViewing.Desc,
				Price:         req.CurrentViewing.GetPriceInt(),
				PriceDisplay:  req.CurrentViewing.GetPriceDisplay(),
				StockQuantity: 50,
				ImageUrl:      req.CurrentViewing.Img,
			}
		}
		if req.CurrentViewing.Name != "" {
			viewProd.Name = req.CurrentViewing.Name
		}
		if req.CurrentViewing.GetPriceDisplay() != "" {
			viewProd.PriceDisplay = req.CurrentViewing.GetPriceDisplay()
		}
		if req.CurrentViewing.GetPriceInt() > 0 {
			viewProd.Price = req.CurrentViewing.GetPriceInt()
		}
		if req.CurrentViewing.Img != "" {
			viewProd.ImageUrl = req.CurrentViewing.Img
		}
		if req.CurrentViewing.Desc != "" {
			viewProd.Desc = req.CurrentViewing.Desc
		}
		if req.CurrentViewing.Category != "" {
			viewProd.Category = req.CurrentViewing.Category
		}
		if vCode := req.CurrentViewing.GetVariantCode(); vCode != "" && viewProd.Code == "" {
			viewProd.Code = vCode
		}
		if vHex := req.CurrentViewing.GetVariantColor(); vHex != "" && viewProd.Hex == "" {
			viewProd.Hex = vHex
		}

		// Lưu vào lịch sử xem hàng của phiên chat (Session Tracking)
		e.sessionMgr.RecordProductView(sessionID, *viewProd)
	}

	// Lấy danh sách sản phẩm khách đã xem trước đó trong cùng phiên chat (không tính sản phẩm hiện tại)
	var prevViewed []model.Product
	if viewProd != nil {
		prevViewed = e.sessionMgr.GetSessionHistory(sessionID, viewProd.ID)
	} else {
		prevViewed = e.sessionMgr.GetSessionHistory(sessionID, 0)
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

	lowerMsg := strings.ToLower(userMsg)
	isCrossSell := strings.Contains(lowerMsg, "hợp không") || strings.Contains(lowerMsg, "kết hợp") ||
		strings.Contains(lowerMsg, "phối") || strings.Contains(lowerMsg, "mua son này") ||
		strings.Contains(lowerMsg, "mua phấn này") || strings.Contains(lowerMsg, "đi kèm")

	if cat == "" && viewProd != nil && viewProd.Category != "" {
		cat = viewProd.Category
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

	isPolicyOrHours := strings.Contains(lowerMsg, "mở cửa") || strings.Contains(lowerMsg, "đóng cửa") ||
		strings.Contains(lowerMsg, "giờ") || strings.Contains(lowerMsg, "đổi trả") ||
		strings.Contains(lowerMsg, "bảo hành") || strings.Contains(lowerMsg, "hoàn tiền") ||
		strings.Contains(lowerMsg, "trả hàng") || strings.Contains(lowerMsg, "lỗi") ||
		strings.Contains(lowerMsg, "gãy") || strings.Contains(lowerMsg, "vỡ")

	// Tìm kiếm sản phẩm phù hợp
	matchedItems := e.repo.SearchProducts(cat, maxP, un, userMsg, 3)

	// TƯ VẤN CHÉO (Cross-sell tự động):
	// Khi khách hỏi "chị đang tính mua son này, có phấn nào hợp không?",
	// Go tra ngay lịch sử xem hàng của khách để gợi ý đúng loại phấn khách vừa ngắm nghía trước đó.
	if (isCrossSell || (cat != "" && viewProd != nil && !strings.EqualFold(cat, viewProd.Category))) && len(prevViewed) > 0 {
		var crossSellItem *model.Product
		for i := len(prevViewed) - 1; i >= 0; i-- {
			p := prevViewed[i]
			if cat == "" || strings.EqualFold(p.Category, cat) {
				crossSellItem = &p
				break
			}
		}
		if crossSellItem != nil {
			var newMatched []model.Product
			newMatched = append(newMatched, *crossSellItem)
			for _, item := range matchedItems {
				if item.ID != crossSellItem.ID && !strings.EqualFold(item.Name, crossSellItem.Name) {
					newMatched = append(newMatched, item)
				}
			}
			if len(newMatched) > 3 {
				newMatched = newMatched[:3]
			}
			matchedItems = newMatched
		}
	} else if viewProd != nil && !isPolicyOrHours {
		// Nếu khách đang xem sản phẩm trên màn hình và không phải câu hỏi thuần policy/giờ làm:
		// Nhận diện câu hỏi mở/ngắn, ưu tiên CurrentViewing và tự động đưa vào vị trí đầu của MatchedItems nếu phù hợp.
		isConflictingCategory := cat != "" && viewProd.Category != "" && !strings.EqualFold(cat, viewProd.Category)
		if !isConflictingCategory {
			newMatched := []model.Product{*viewProd}
			for _, item := range matchedItems {
				if item.ID != viewProd.ID && !strings.EqualFold(item.Name, viewProd.Name) {
					newMatched = append(newMatched, item)
				}
			}
			if len(newMatched) > 3 {
				newMatched = newMatched[:3]
			}
			matchedItems = newMatched
		}
	}

	var viewingContext strings.Builder
	if viewProd != nil {
		viewingContext.WriteString(fmt.Sprintf("\n[SẢN PHẨM KHÁCH ĐANG XEM TRÊN MÀN HÌNH]:\n- Tên: %s | Giá: %s | Phân loại: %s | Chi tiết: %s\n- LƯU Ý BẮT BUỘC: Khách hàng đang trực tiếp nhìn thấy sản phẩm này trên màn hình Miniapp/Web. Khi khách hỏi câu ngắn, cụt ngủn hoặc ngầm định như 'có mẫu không?', 'có mẫu thử không em?', 'cách dùng thế nào?', 'màu này hợp da ngăm không?', 'có tốt không?', AI BẮT BUỘC hiểu ngay 100%% là khách đang hỏi về sản phẩm này. Khóa ngữ cảnh và ưu tiên trả lời chi tiết về sản phẩm này.\n",
			viewProd.Name, viewProd.PriceDisplay, viewProd.Category, viewProd.Desc))
	}

	if len(prevViewed) > 0 {
		viewingContext.WriteString("\n[LỊCH SỬ SẢN PHẨM KHÁCH ĐÃ XEM TRONG PHIÊN NÀY]:\n")
		for _, p := range prevViewed {
			viewingContext.WriteString(fmt.Sprintf("- Tên: %s | Giá: %s | Phân loại: %s\n", p.Name, p.PriceDisplay, p.Category))
		}
		viewingContext.WriteString("- LƯU Ý TƯ VẤN CHÉO (CROSS-SELL): Khi khách hỏi 'chị đang tính mua son này, có phấn nào hợp không?' hoặc hỏi tư vấn chéo, hãy tra ngay lịch sử xem hàng trên để gợi ý đúng sản phẩm khách vừa ngắm nghía trước đó.\n")
	}

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
5. Trả lời súc tích, dưới 90 chữ.` + viewingContext.String() + catalogContext.String()

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
