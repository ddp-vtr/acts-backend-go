# Bảng tổng hợp số liệu chắc chắn — Go + Multi-Agent (ActsOne)

Dùng cho vòng 2 ngày 06/11/2026. Chỉ gồm số liệu đã qua kiểm chứng hoặc
được 2–3 bản nghiên cứu độc lập cùng ghi và dẫn tài liệu chính thức.
Mọi con số chưa chắc đã tách riêng xuống cuối — không trích khi phỏng vấn.

## Cách đọc

- ✅ **Đã kiểm chứng trực tiếp** — Muse tự mở đọc tài liệu gốc. Trích thoải mái.
- 🟡 **Nhiều bản cùng ghi + dẫn doc chính thức** — Muse chưa tự mở đọc,
  nhưng 2–3 bản nghiên cứu độc lập ghi giống nhau và đều dẫn tài liệu
  chính thức của sàn / blog kỹ thuật chính thức. Dùng được, nói rõ
  "theo tài liệu của [sàn]".
- 🔴 **Chưa chắc chắn** — mâu thuẫn giữa các bản, nguồn thứ cấp, hoặc
  mô hình giả định. KHÔNG trích khi phỏng vấn.

---

## ✅ Bảng 1 — Đã kiểm chứng trực tiếp

| # | Số liệu | Nguồn |
|---|---------|-------|
| 1 | **Lazada push: trả HTTP 200 trong 500ms** | Lazada Push Mechanism doc, mở trực tiếp 05/11/2026. Nguyên văn: "ack with Http status code 200 within 500ms". |
| 2 | **Lazada retry: mỗi 30 phút, dừng sau hơn 12 lần thất bại** | Cùng doc trên. Nguyên văn: retry sau 30 phút; "if you fail more than 12 times... server will stop retrying". |
| 3 | **Lazada chữ ký push: Hex-encoded HMAC-SHA256, khóa = appSecret, băm body** | Cùng doc trên. |
| 4 | **Lazada lỗi 998: giới hạn lưu lượng, thường xảy ra đầu Mega Campaign; xem thông báo downgrade trước Campaign (ví dụ: LAZADA D6 Campaign Downgrade Plan)** | Lazada "Create/Update Product Q&A" doc, mở trực tiếp 05/11/2026. |
| 5 | **Lazada lỗi 501: SKU tham gia Campaign bị khóa một phần tồn kho bán được; lỗi xảy ra khi cập nhật tồn kho khả dụng xuống thấp hơn mức đang bị khóa ("negative sellable on channel inventory"); SKU đang chạy campaign cũng không sửa được thông tin đến khi campaign kết thúc** | Cùng doc trên (Q5, Q8). |
| 6 | **Lazada ApiCallLimit: giới hạn tần suất theo từng API; GetBrandByPages giới hạn 30 QPS; vượt ngưỡng bị chặn 1 giây rồi thử lại** | Cùng doc trên (Q9, Q10). |
| 7 | **Shopee chữ ký push: HMAC-SHA256 trên `URL\|raw_body`, khóa Partner Key, hex-encode; không parse JSON rồi serialize lại trước khi băm** | Shopee Push Mechanism doc, mở trực tiếp 05/11/2026. Trang dùng từ "URL" và "response.content". |
| 8 | **Shopee push: phải trả 2xx kèm empty body; thiếu 1 trong 2 thì tính là Push lỗi** | Cùng doc trên. |
| 9 | **Shopee cảnh báo/khóa webhook — cả 2 điều kiện phải đồng thời xảy ra: hơn 600 push trong 6 giờ qua VÀ tỷ lệ thành công nhỏ hơn 70% → email cảnh báo mỗi 30 phút; nhỏ hơn 30% → khóa đăng ký push. Thông báo bị lỡ trong thời gian bị khóa sẽ không được gửi bù; tỷ lệ thành công tính lại từ đầu khi đăng ký mới** | Cùng doc trên. Lưu ý: "more than 600" (hơn 600, không phải đúng 600), "less than 70%/30%" (nhỏ hơn, không phải nhỏ hơn hoặc bằng). |
| 10 | **Luật Bảo vệ dữ liệu cá nhân 91/2025/QH15, hiệu lực 01/01/2026; Điều 20: xử lý dữ liệu VN ở nước ngoài = chuyển dữ liệu xuyên biên giới, phải lập hồ sơ đánh giá tác động + nộp 1 bản trong 60 ngày** | Kiểm chứng 27/10/2026. |
| 11 | **Nghị định 89/2024/NĐ-CP là về chuyển đổi công ty nhà nước thành TNHH 1 thành viên — KHÔNG liên quan TikTok Shop hay KOL** | Kiểm chứng 05/11/2026 (bản "Khung Tuân thủ KOL TikTok Shop" không tồn tại). |

---

## 🟡 Bảng 2 — Nhiều bản cùng ghi + dẫn doc chính thức

### TikTok Shop (theo TikTok Shop Partner docs)

| # | Số liệu | Ghi chú |
|---|---------|---------|
| 12 | Timeout: trả HTTP 200 trong 3 giây (3.000ms) | Bản 2 + bản 3 cùng ghi. Trang Partner Center yêu cầu đăng nhập nên Muse chưa mở được nội dung doc trực tiếp. |
| 13 | Chữ ký: HMAC-SHA256 trên `{app_key}{raw_payload}`, khóa app_secret, đối chiếu header Authorization | Bản 2 + bản 3 cùng ghi. Cùng lý do như trên. |
| 14 | Dừng retry sau lần thất bại thứ 4 | Bản 2 + bản 3 cùng ghi. Cùng lý do như trên. |

### Tiki (theo Tiki Open API docs)

| # | Số liệu | Ghi chú |
|---|---------|---------|
| 15 | Chữ ký HMAC-SHA1, đặt trong header x-signature | Bản 1 + bản 2 + bản 3 cùng ghi. |

### Mô hình & kiến trúc (từ blog kỹ thuật chính thức, 2–3 bản cùng ghi)

| # | Số liệu | Ghi chú |
|---|---------|---------|
| 16 | Shopify Sidekick: "Agents propose, merchants approve"; giữ mỗi agent dưới 20 tool; trên 50 tool thì loạn ("Death by a Thousand Instructions"); dùng JIT Instructions | Shopify Engineering blog. Bản 2 + bản 3 cùng ghi. |
| 17 | Cainiao: thuật toán PITO-L (TAPF/MAPF) tăng 10–11% công suất phân loại; Decision-making AI tách khỏi Generative AI | Alibaba Cloud blog + bài báo INFORMS. Bản 2 + bản 3 cùng ghi. |
| 18 | Uber Cadence: Workflow tất định + Activity chứa side-effect; Temporal kế thừa Cadence | Uber Engineering blog + Temporal docs. Bản 2 + bản 3 cùng ghi. |
| 19 | Ozon mở mã nguồn Go: file.d, seq-db, insane-json, cute, testo; pg_doorman (Rust) làm connection pooler | github.com/ozontech — repo công khai, kiểm chứng được. Bản 3 ghi. |
| 20 | ByteDance: framework Eino (Go) cho LLM Agent; CloudWeGo gồm Kitex (RPC) + Hertz (HTTP) | github.com/cloudwego/eino — repo công khai. Bản 2 + bản 3 cùng ghi. |
| 21 | Token sống: Shopee access_token 4 giờ; Lazada access_token ~30 ngày (refresh 180 ngày); TikTok Shop access_token 90 ngày | Chỉ bản 3 ghi — dùng được nhưng đừng nhấn mạnh. |

### Pháp luật Việt Nam (văn bản chính thức)

| # | Số liệu | Ghi chú |
|---|---------|---------|
| 22 | NĐ 13/2023 Điều 25: chuyển dữ liệu cá nhân ra nước ngoài phải lập hồ sơ đánh giá tác động | Văn bản chính thức. |
| 23 | Luật Dữ liệu 60/2024/QH15, hiệu lực 01/07/2025; Điều 23 về chuyển/xử lý dữ liệu xuyên biên giới | Văn bản chính thức. |

---

## 🔴 Bảng 3 — KHÔNG trích khi phỏng vấn

| Số liệu | Vì sao chưa chắc |
|---------|------------------|
| Wildberries: >10 triệu hàng/ngày (bản 2) vs 25 triệu đơn + 1 tỷ requests/ngày (bản 3) | Hai bản mâu thuẫn nhau. |
| ByteDance: 70% microservices viết bằng Go | Nguồn thứ cấp (blog datasumi); bản 3 dẫn go.dev case studies nhưng chưa tự mở kiểm chứng. |
| Uber: 2.100–3.000 microservices / ~50 triệu dòng code Go | Nguồn thứ cấp (Scribd, blog tổng hợp). |
| Chi phí token 1,5–3 USD/ngày; 200 ca LLM/ngày; 98% hot path / 2% ngoại lệ | Mô hình giả định trong bản nghiên cứu, chưa đo thật. |
| Latency budget P99 69ms / tổng <50ms / P99 <70ms | Mô hình, chưa đo thật. |
| Lộ trình Tháng 1–9 / kế hoạch 90 ngày đầu | Đề xuất của bản nghiên cứu, không phải kế hoạch thật của ActsOne. |
| JD.com: tồn kho 30,3 ngày, 1.300 kho, 43 trung tâm Asia No.1 | Chỉ bản 3 ghi, chưa đối chiếu bản khác. |

---

## Cách nói an toàn khi phỏng vấn

- Số ✅: nói thẳng kèm nguồn. Ví dụ: "Theo tài liệu Push Mechanism
  của Lazada, endpoint phải trả HTTP 200 trong 500ms, retry mỗi 30 phút
  và dừng sau hơn 12 lần thất bại."
- Số 🟡: nói "Theo tài liệu chính thức của [sàn]..." — đúng, vì các bản
  nghiên cứu đều dẫn doc chính thức.
- Số 🔴: đừng nhắc con số. Nếu bị hỏi, nói: "Tôi chưa kiểm chứng con số
  cụ thể, nhưng nguyên tắc là..." rồi nói về nguyên tắc.

---

*Tổng hợp từ 3 bản nghiên cứu Go + Agent ngày 05/11/2026 (v1, v2, v3)
và các lần kiểm chứng trực tiếp của Muse. Cập nhật 05/11/2026 16:08:
Shopee đã kiểm chứng xong từ doc chính thức và chuyển lên Bảng 1 (kèm
chi tiết: 2 điều kiện cảnh báo/khóa phải đồng thời xảy ra; "hơn 600",
"nhỏ hơn 70%/30%"). TikTok vẫn ở tầng 🟡 vì Partner Center yêu cầu
đăng nhập, chưa mở được nội dung doc trực tiếp.*
