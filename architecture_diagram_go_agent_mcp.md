# SƠ ĐỒ VẬN HÀNH KIẾN TRÚC HYBRID: GO (HOT PATH) + MULTI-AGENT + MCP + OMNICHANNEL

> **Hệ thống:** Omnichannel OMS / WMS / ERP  
> **Nền tảng tích hợp:** TikTok Shop, Shopee, Facebook / Messenger, Web Brand  
> **Công nghệ lõi:** Golang (High-Speed Ingress), Redis Stream, MCP Server, Multi-Agent Orchestrator, PostgreSQL/MySQL.

---

## 1. SƠ ĐỒ KIẾN TRÚC TỔNG THỂ (MERMAID)

```mermaid
flowchart TD
    %% EXTERNAL PLATFORMS
    subgraph Platforms["1. Sàn Thương Mại Điện Tử & Mạng Xã Hội"]
        TK["TikTok Shop<br/>(Webhook HMAC-SHA256 / REST)"]
        SP["Shopee Open Platform<br/>(Webhook / REST API v2)"]
        FB["Facebook / Messenger<br/>(Graph API / Webhook)"]
    end

    %% HOT PATH (GO HIGH-SPEED CORE)
    subgraph HotPath["2. HOT PATH: Go High-Speed Core (Latency < 50ms, 100% Deterministic)"]
        GW["Go API Gateway / Ingress<br/>- Verify Signature<br/>- IP Whitelisting & Rate Limit"]
        IDEMP["Idempotency Gate<br/>- Redis SETNX (lock:event_id, TTL=24h)<br/>- Chống duplicate webhook"]
        
        subgraph GoOMS["Go OMS / WMS Engine"]
            FSM["Order Finite State Machine<br/>(CREATED -> PAID -> FULFILLED / CANCEL)"]
            TX["Database Transaction Isolation<br/>(SELECT stock FOR UPDATE)"]
        end
        
        DB[("Database (PostgreSQL / MySQL)<br/>- Orders, Inventory, Ledgers<br/>(Single Source of Truth)")]
        STREAM[("Event Bus (Redis Stream / Kafka)<br/>- order.created, order.failed<br/>- stock.depleted, chat.inbound")]
    end

    %% ASYNC CONTROL PLANE (AGENT + MCP)
    subgraph ControlPlane["3. CONTROL PLANE: Async Multi-Agent (LLM Reasoning Layer)"]
        CONSUMER["Go Event Consumer & Dispatcher<br/>- Lọc đơn lỗi DLQ<br/>- Lọc tin nhắn CSKH Facebook"]
        MASK["PII Data Masking Engine<br/>- Mã hóa SĐT, Tên, Địa chỉ<br/>- Tuân thủ Nghị định 13/2023/NĐ-CP"]
        
        subgraph AgentBrain["Multi-Agent Decision Orchestrator"]
            AG_DLQ["Exception & DLQ Agent<br/>(Phân loại lỗi, đề xuất retry/hủy)"]
            AG_CSKH["CSKH & Consultation Agent<br/>(Soạn phản hồi Messenger/Chat Sàn)"]
            AG_RECON["Reconciliation Agent<br/>(Đối soát số liệu kho & sàn định kỳ)"]
        end
        
        subgraph MCPLayer["Go MCP Server (Model Context Protocol)"]
            MCP_READ["Read-Only Tools<br/>- get_order_detail(order_id)<br/>- check_inventory(sku_code)<br/>- get_carrier_status(tracking_no)"]
            MCP_PROP["Proposal Tool (Write Gate)<br/>- create_draft_proposal(...)"]
        end
        
        LLM["LLM Inference Provider<br/>(Claude 3.5 Sonnet / Gemini 1.5 Pro)"]
    end

    %% GOVERNANCE & EXECUTION
    subgraph Governance["4. GOVERNANCE: Human-in-the-Loop & Execution Actuator"]
        DRAFT_DB[("Draft Proposals Table<br/>- status: PENDING_APPROVAL<br/>- payload_action, reasoning_log")]
        DASHBOARD["Admin Web Dashboard<br/>(Vận hành viên / Lead duyệt 1-Click)"]
        ACTUATOR["Go Write Actuator<br/>(Thực thi vật lý khi có chữ ký duyệt:<br/>Update DB, Call API Sàn, Trừ kho)"]
    end

    %% PIPELINE CONNECTIONS
    %% Ingress to Go
    TK -->|Webhook| GW
    SP -->|Webhook| GW
    FB -->|Inbound Webhook| GW

    %% Hot Path processing
    GW -->|1. Xác thực HMAC| IDEMP
    IDEMP -->|2. Key hợp lệ, chưa xử lý| FSM
    IDEMP -.->|Duplicate event: Trả 200 OK ngay| GW
    FSM -->|3. Atomic Lock & Cập nhật kho| TX
    TX --> DB
    GW -.->|Phản hồi 200 OK trong < 50ms| Platforms
    TX -->|4. Publish Event bất đồng bộ| STREAM

    %% Async processing to Agent
    STREAM -->|Consume events| CONSUMER
    CONSUMER -->|Event bất thường / DLQ / CSKH| MASK
    MASK --> AG_DLQ & AG_CSKH & AG_RECON

    %% Agent reasoning & MCP interaction
    AG_DLQ & AG_CSKH & AG_RECON <-->|Gửi prompt & nhận phân tích| LLM
    AG_DLQ & AG_CSKH & AG_RECON -->|Tra cứu dữ liệu sạch| MCP_READ
    MCP_READ -->|Truy vấn Read-Only| DB

    %% Proposal & Human approval
    AG_DLQ & AG_CSKH -->|Tạo đề xuất giải pháp| MCP_PROP
    MCP_PROP -->|Ghi bản ghi đề xuất| DRAFT_DB
    DRAFT_DB -->|Hiển thị danh sách chờ duyệt| DASHBOARD
    DASHBOARD -->|Vận hành viên bấm Duyệt (Approve)| ACTUATOR
    ACTUATOR -->|Ghi nhận số liệu chính thức| DB
    ACTUATOR -.->|Gọi API sàn: Hủy đơn / Hoàn tiền / Trả lời khách| Platforms

    %% STYLING
    classDef hotpath fill:#ffebee,stroke:#c62828,stroke-width:2px;
    classDef agent fill:#e8eaf6,stroke:#283593,stroke-width:2px;
    classDef gov fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px;
    class GW,IDEMP,GoOMS,DB,STREAM hotpath;
    class AG_DLQ,AG_CSKH,AG_RECON,MCPLayer,LLM agent;
    class DRAFT_DB,DASHBOARD,ACTUATOR gov;
```

---

## 2. BÓC TÁCH CHI TIẾT 4 PHÂN VÙNG KIẾN TRÚC

### Vùng 1: Ingress Platforms (TikTok Shop, Shopee, Facebook)
- **TikTok Shop & Shopee:** Đẩy Webhook khi có sự kiện `order.created`, `order.paid`, `order.cancelled`. Yêu cầu SLA phản hồi HTTP 200 trong vòng 2.000ms (TikTok) và 3.000ms (Shopee).
- **Facebook / Messenger:** Nhận tin nhắn khách hỏi về sản phẩm, tình trạng đơn hàng, khiếu nại qua Graph API Webhook.

### Vùng 2: Hot Path (Go High-Speed Core)
- **Đặc tính:** 100% Deterministic (tính toán xác định), latency siêu thấp (< 50ms), không có LLM can thiệp.
- **Go API Gateway:** Kiểm tra chữ ký HMAC-SHA256 từ webhook secret của sàn.
- **Idempotency Gate:** Dùng Redis `SETNX lock:event_id 1 EX 86400`. Nếu key đã tồn tại $\rightarrow$ trả ngay 200 OK để chống sàn retry trùng đơn.
- **Go OMS/WMS Engine:** Chạy State Machine (FSM), dùng Database Transaction Isolation (`SELECT ... FOR UPDATE`) để trừ kho an toàn, chống overselling (bán quá số lượng tồn).
- **Event Bus:** Đẩy sự kiện thành công hoặc thất bại vào Redis Stream / Kafka.

### Vùng 3: Control Plane (Multi-Agent + MCP Server)
- **Đặc tính:** Chạy bất đồng bộ (Async Worker), chịu trách nhiệm suy luận logic và xử lý ngoại lệ.
- **PII Data Masking:** Go bóc tách các trường nhạy cảm (Họ tên, SĐT, Địa chỉ nhà) trước khi gửi qua API LLM bên thứ ba (Tuân thủ Nghị định 13/2023/NĐ-CP).
- **Go MCP Server:** Đóng vai trò là cầu nối giao tiếp chuẩn hóa Model Context Protocol:
  - `Read-Only Tools`: Cho phép Agent đọc trạng thái đơn, tồn kho, mã vận đơn từ DB.
  - `Proposal Tool`: Agent **không được phép ghi đè** DB hay gọi API sàn trực tiếp, chỉ được gọi `create_draft_proposal`.

### Vùng 4: Governance & Execution (Human-in-the-Loop)
- **Draft Proposals Table:** Lưu trữ đề xuất của Agent kèm lý do suy luận (Reasoning Log) và payload dự kiến.
- **Dashboard:** Vận hành viên hoặc IT Lead kiểm tra và bấm "Phê duyệt" (Approve) hoặc "Từ chối" (Reject).
- **Go Write Actuator:** Nhận tín hiệu phê duyệt $\rightarrow$ Thực thi lệnh ghi vật lý vào DB nội bộ và gọi REST API sàn tương ứng.

