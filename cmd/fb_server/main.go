package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"acts-backend-go/internal/rag"
)

var (
	verifyToken     = "demo-verify-token"
	appSecret       = "5ffd43248abc60030a1d51e5ba9278a4"
	pageAccessToken = "EAA5kZCxbxWOkBS3ajeYMKomAvywf4XsavOrB5jLSqduJX0U9EiWBUj6RqOq3VOMON5XEtBPsDNTmHiBlcZAZAKQgPEN2A4ktVwKqSh5pYAaFQ7TP2lZBZCV4Di2Tn3FobvHy1xNRKB0nEdar57IOZAyu7tZBQl1LVUi9ZAk1HhxoJRHhJQXSsqNuBqjJQ1F1HSrjbCdzLOZAqqgZDZD"
	ragEngine       *rag.Engine
)

type WebhookEvent struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Message struct {
				MID  string `json:"mid"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

func verifySignature(payload []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedHash := strings.ToLower(signature[7:])
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(payload)
	actualHash := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}

// Gửi tin nhắn text thông thường
func sendTextMessage(recipientID, text string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/me/messages?access_token=%s", pageAccessToken)
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": text},
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FB API error: %s", string(respBody))
	}
	return nil
}

// Gửi khung Generic Template (Thẻ Carousel sản phẩm kèm ảnh và nút xem web)
func sendProductCarousel(recipientID string, items []model.Product) error {
	if len(items) == 0 {
		return nil
	}
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/me/messages?access_token=%s", pageAccessToken)

	var elements []map[string]interface{}
	for i, item := range items {
		if i >= 4 { // Messenger cho tối đa 10 thẻ, lấy 3-4 thẻ đẹp nhất
			break
		}
		imgUrl := item.ImageUrl
		if imgUrl == "" {
			imgUrl = "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-VELVET-TINT-RENEWAL-1.png"
		}
		elem := map[string]interface{}{
			"title":     item.Name,
			"image_url": imgUrl,
			"subtitle":  fmt.Sprintf("%s - %s", item.PriceDisplay, item.Desc),
			"buttons": []map[string]interface{}{
				{
					"type":  "web_url",
					"url":   "https://miniapp.ddphuong.id.vn/",
					"title": "Xem Sản Phẩm",
				},
			},
		}
		elements = append(elements, elem)
	}

	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "template",
				"payload": map[string]interface{}{
					"template_type": "generic",
					"elements":      elements,
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			log.Printf("[Webhook] Verified subscription challenge: %s", challenge)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}

		sig := r.Header.Get("X-Hub-Signature-256")
		if sig != "" && !verifySignature(body, sig) {
			log.Printf("[Webhook] Invalid signature")
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}

		var event WebhookEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "Error parsing json", http.StatusBadRequest)
			return
		}

		if event.Object == "page" {
			for _, entry := range event.Entry {
				for _, msg := range entry.Messaging {
					senderID := msg.Sender.ID
					text := msg.Message.Text
					if text != "" {
						log.Printf("[Messenger] Received from %s: %s", senderID, text)
						go func(sID, userMsg string) {
							// Call Go RAG Engine
							ragResp, err := ragEngine.ProcessQuery(model.ChatRequest{Message: userMsg})
							replyText := "Dạ em chào Chị ạ, em là tư vấn viên của SKY. Chị đang quan tâm dòng son hay phấn nào để em hỗ trợ nhé!"
							if err == nil && ragResp.Reply != "" {
								replyText = ragResp.Reply
							}
							// 1. Gửi tin nhắn text tư vấn
							if err := sendTextMessage(sID, replyText); err != nil {
								log.Printf("[Messenger] Error sending text: %v", err)
							}

							// 2. Nếu có sản phẩm gợi ý -> Gửi kèm Thẻ Carousel sản phẩm có ảnh + nút xem web
							if err == nil && len(ragResp.MatchedItems) > 0 {
								if err := sendProductCarousel(sID, ragResp.MatchedItems); err != nil {
									log.Printf("[Messenger] Error sending carousel: %v", err)
								} else {
									log.Printf("[Messenger] Sent product carousel (%d items) to %s", len(ragResp.MatchedItems), sID)
								}
							}
						}(senderID, text)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("EVENT_RECEIVED"))
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// REST API endpoint cho Miniapp / Web / Kênh khác gọi
func chatApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, HTTP-Referer, X-Title")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resp, err := ragEngine.ProcessQuery(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	if token := os.Getenv("VERIFY_TOKEN"); token != "" {
		verifyToken = token
	}
	if secret := os.Getenv("APP_SECRET"); secret != "" {
		appSecret = secret
	}
	if pat := os.Getenv("PAGE_ACCESS_TOKEN"); pat != "" {
		pageAccessToken = pat
	}

	repo := database.NewRepository()
	endpoint := "https://rlau4p8.abc-tunnel.us/v1/chat/completions"
	apiKey := "sk-5b2c9a975f7f309f-qcuwzl-e7e1385d"
	modelName := "ag/gemini-2.5-flash"

	if ep := os.Getenv("AI_ENDPOINT"); ep != "" {
		endpoint = ep
	}
	if key := os.Getenv("AI_API_KEY"); key != "" {
		apiKey = key
	}
	if m := os.Getenv("AI_MODEL"); m != "" {
		modelName = m
	}

	ragEngine = rag.NewEngine(repo, endpoint, apiKey, modelName)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"online","service":"acts-backend-go","version":"1.0.0"}`))
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/api/chat", chatApiHandler)

	port := normalizePort(os.Getenv("PORT"))

	if port != "8081" {
		go func() {
			log.Printf("Secondary listener active on port 8081 for reverse-proxy fallback...")
			if err := http.ListenAndServe(":8081", nil); err != nil {
				log.Printf("Secondary listener on 8081 stopped: %v", err)
			}
		}()
	}

	log.Printf("Server Go RAG listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func normalizePort(port string) string {
	port = strings.TrimSpace(port)
	port = strings.TrimPrefix(port, ":")
	if port == "" {
		return "8081"
	}
	return port
}
