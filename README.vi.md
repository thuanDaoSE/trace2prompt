<div align="center">
  
  # 🚀 Trace2Prompt
  **Trợ lý Debug AI "Zero-Config" - Tự động gom trọn Runtime Context & Log phân tán**
  
  [![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/trace2prompt)](https://goreportcard.com/report/github.com/yourusername/trace2prompt)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
</div>

## 😩 Nỗi đau: "AI ơi, tại sao app của tôi sập?"

_Đọc tài liệu này bằng ngôn ngữ khác: [ENGLISH](README.md)._

Bạn mở ChatGPT/Claude lên và gõ:

> _"Ê AI, tôi bấm nút A, rồi điền form B, tự nhiên dự án không chạy, tại sao chỗ này lại lỗi nghiệp vụ? Hệ thống chậm rì là do đâu?"_

Và kết quả? AI trả lời những câu chung chung sáo rỗng, hoặc tệ hơn là **bịa ra code sai**. Lý do đơn giản là vì **AI bị mù Môi trường Runtime (Ngữ cảnh lúc chạy)**. Nó chỉ biết đọc code tĩnh, chứ không biết dữ liệu thực tế lúc đó là gì.

Hơn nữa, trong các hệ thống hiện đại, **Log thường văng tứ tung mỗi nơi một nẻo**: Frontend báo lỗi ở Console trình duyệt, Backend văng Exception ở Terminal, SQL thì kẹt ở Database.
Để AI hiểu, bạn phải lóc cóc đi chắp vá thủ công từ 3-4 nơi khác nhau. Việc đi gom nhặt đống log phân tán này cực kỳ tốn thời gian và làm anh em Dev "lười" dùng AI để debug lỗi phức tạp!

## 🎬 See it in Action (27s Demo)

https://github.com/user-attachments/assets/4ee35c7e-06fd-4695-acd6-b6e109975786

## 💡 Giải pháp: Trace2Prompt

**Trace2Prompt** là một công cụ chạy ngầm (Daemon) cực nhẹ, đóng vai trò như một trạm thu thập dữ liệu chuẩn OpenTelemetry (OTLP).

Thay vì phải lười biếng đi gom log thủ công, **chỉ với 1 click chuột**, Trace2Prompt sẽ tự động tóm gọn toàn bộ **Runtime Context (Hành trình xuyên suốt)**:

- 🖱️ Hành vi của User (Bấm vào đâu, gọi API nào trên Frontend).
- 🌐 Trạng thái HTTP & Headers (ẩn Token bảo mật tự động).
- ⚙️ Flame Graph thực thi dưới Backend (Lỗi ở class nào, dòng code nào).
- 🗄️ Các câu lệnh SQL đã được chạy thực tế.
- 🐰 Luồng chạy ngầm Async (RabbitMQ/Kafka) nếu có.

...và đóng gói tất cả đống log phân tán đó thành một **Prompt chuẩn mực 100%**, sẵn sàng để ném cho AI "bắt bệnh" chính xác tới từng dòng code!

## 🗺️ Sơ đồ hoạt động (Workflow)

```mermaid
sequenceDiagram
    actor Dev as User / Dev
    participant FE as Frontend (trace2prompt.js)
    participant BE as Backend (OTel Agent)
    participant DB as DB / Message Queue
    participant T2P as Trace2Prompt
    participant AI as AI (Cursor/Claude)

    Dev->>FE: 1. Thao tác click (VD: Thanh toán)

    rect rgba(128, 128, 128, 0.15)
    Note over FE, T2P: Quá trình Code chạy & Trace2Prompt thu thập ngầm
    FE->>BE: 2. Gọi HTTP API (Tự động gắn TraceID)
    BE->>DB: 3. Query SQL / Publish Message
    DB-->>BE: 4. Xảy ra lỗi (Exception / Timeout)
    BE-->>FE: 5. Trả về HTTP 500

    FE-->>T2P: 6. Gửi UI Journey (Click, Console, Headers)
    BE-->>T2P: 7. Gửi OTLP (Traces, Logs, Flame Graph)
    T2P->>T2P: 8. Nối vết E2E Context & Che Token bảo mật
    end

    alt Chế độ Thủ công (Web UI)
        Dev->>T2P: 9a. Copy Prompt từ Web UI (Cổng 4319)
        Dev->>AI: 10a. Dán Prompt vào ChatGPT/Claude
    else Chế độ Tự động (MCP)
        AI->>T2P: 9b. Cursor tự gọi Tool MCP lấy Context
    end

    AI-->>Dev: 11. Chỉ ra gốc rễ lỗi & Đưa code fix chuẩn xác!
```

## ✨ Tính năng nổi bật

- **⚡ Zero-Config (Không cần sửa code):** Cắm Agent vào lệnh chạy app thông thường là có thể bắt đầu monitor.
- **🪶 Siêu nhẹ & Tối ưu (Low Footprint):** Được viết bằng Golang, công cụ chạy ngầm cực kỳ êm ái, gần như không tốn CPU và **chỉ chiếm vài chục MB RAM**. Không làm chậm máy của bạn!
- **🚀 Tăng hiệu suất Debug với AI gấp 10 lần:**
  - **Gộp Log E2E:** Không còn cãi nhau xem lỗi do Front hay Back. Tool gom trọn Console/Click của Frontend + API Backend + Background System Errors thành 1 luồng duy nhất.
  - **Soi thấu Database:** Cung cấp chi tiết Flame Graph thứ tự chạy và trích xuất nguyên bản các câu lệnh SQL. AI sẽ nhìn vào đó để bắt ngay các lỗi N+1 Query hoặc Deadlock.
- **🛡️ Đề cao Bảo mật (Privacy First):** Các thông tin nhạy cảm như Password, JWT Token, Email tự động bị băm nát thành `[REDACTED]` trước khi đưa cho AI.
- **🤖 Tích hợp AI Agents (MCP):** Hỗ trợ giao thức MCP cho phép các AI IDE (như Cursor) tự động rút trích ngữ cảnh mà không cần Copy-Paste.

## 🎯 Ví dụ Output (Cục Prompt thực tế gửi cho AI)

![Trace2Prompt UI](assets/img_demo.png)

Trace2Prompt sẽ sinh ra một cấu trúc hoàn hảo như sau (bạn chỉ việc Copy & Paste):

```text
Please analyze the system error based on the E2E Runtime Context below:

=================================================
TraceID: `6464d81b63cbc1de7184d0c90ce53891`

### 🖥️ ENVIRONMENT & INFRASTRUCTURE
- Service: `coffee-order-app`
- OS: `Windows 11 10.0`
- Runtime: `17.0.12+8-LTS-286`
- Database: `redis @ localhost:8080`
- 📊 CPU Usage (At request time): `0.02%`
- 🧠 JVM Memory Used: `18 MB`

### 🌐 HTTP REQUEST CONTEXT
- Method: `GET`
- URL: `/api/v1/products?search=&page=0&size=6`
- Status Code: `200`
- 🔐 Backend Received Auth: `[TOKEN/COOKIE ATTACHED]`

- 🍪 **Attached Cookies (Keys only):** `[["jwt, refreshToken, i18next"]`

### 👣 FRONTEND JOURNEY (USER JOURNEY)
- [22:27:49] 🖱️ `CLICK` at `http://localhost:5174/` (Element: `[Menu] A.px-3.py-2.rounded-md.text-sm.font-medium.transition-colors.duration-200.text-amber-200.hover:text-amber-50.hover:bg-amber-800/50.`)
- [22:27:30] 🌐 `FRONTEND API CALL` `GET http://localhost:8080/api/v1/products?search=&page=0&size=6` -> Status: `200`
  - 📍 Current Page: `http://localhost:5174/`
  - 📶 Network: `Online` | 🖥️ Screen: `1134x799`
  - 💻 Browser: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36`
  - 🎫 Headers: `{"Accept":"application/json, text/plain, */*"}`
  - 🔺 Response Body:
    {
  "content": [
    {
      "id": 1,
      "name": "Espresso",
      "description": "Rich and pure espresso shot",
      "imageUrl": "/Espresso.png",
      "category": {
        "id": 2,
        "name": "Espresso"
      },
      "variants": [
        {
          "id": 1,
          "sku": "S-29000",
          "size": "S",
          "price": 29000,
          "stockQuantity": null,
          "isActive": true
        },
        {
          "id": 2,
          "sku": "M-35000",
          "size": "M",
          "price": 35000,
          "stockQuantity": null,
          "isActive": true
        },
        {
          "id": 3,
          "sku": "L-39000",
          "size": "L",
          "price": 39000,
          "stockQuantity": null,
          "isActive": true
        }
      ],
      "isActive": true
    },
    {
      "id": 2,
      "name": "Cappuccino",
      "description": "Espresso topped with foamy milk",
      "imageUrl": "/Cappuccino.png",
      "category": {
        "id": 5,
        "name": "Cappuccino"
      },
      "variants": [
        {
          "id": 4,
          "sku": "S-35000",
          "size": "S",
          "price": 35000,
          "stockQuantity": null,
          "isActive": true
        },
        {
          "id": 5,
          "sku": "M-42000",
          "size": "M",
          "price": 42000,
          "stockQuantity": null,
          "isActive": true
        },
        {
          "id": 6,
          "sku": "L-48000",
          "size": "L",
          "price


    ... [TRUNCATED DUE TO LENGTH]
- [22:27:30] 🖱️ `CLICK` at `http://localhost:5174/checkout` (Element: `[The Coffee Corner] SPAN.text-xl.font-bold.text-amber-50`)
- [22:24:11] 🖱️ `CLICK` at `http://localhost:5174/checkout` (Element: `[PLACE ORDER & PAYMENT] BUTTON.w-full.bg-amber-600.text-white.py-3.5.rounded-lg.font-bold.text-lg.shadow-lg.hover:bg-amber-700.hover:shadow-xl.transition-all.disabled:opacity-50.disabled:cursor-not-allowed`)

### 🛤️ BACKEND JOURNEY (LOGS)
- [INFO] [CustomUserDetailsService] User [EMAIL_HIDDEN] has authorities: [ROLE_STAFF]
- [INFO] [ProductController] Calling getAllProducts with search: , page: 0, size: 6

### 🛑 BACKEND EXCEPTION STACKTRACE
- (Backend did not throw Exception)

### ⏳ EXECUTION ORDER & SQL (FLAME GRAPH)
- [35 ms] ⚙️ `GET /api/v1/products`
  - [9 ms] ⚙️ `UserRepository.findByEmail`
    - [8 ms] ⚙️ `SELECT com.coffeeshop.backend.entity.User`
      - [3 ms] 🗄️ [DB] `testdb`
      - [1 ms] 🗄️ [DB] `SELECT testdb.users`
        - Query: `select u1_0.id,u1_0.created_at,u1_0.email,u1_0.fullname,u1_0.password,u1_0.phone,u1_0.role,u1_0.store_id,u1_0.updated_at
        FROM users u1_0
        WHERE u1_0.email=?`
  - [3 ms] ⚡ [REDIS] `GET`
    - Redis Command: `GET products::SimpleKey [, Page request [number: 0, size 6, sort: UNSORTED]]`
=================================================


```

## 🚀 Khởi chạy nhanh (Chỉ 2 phút)

### Bước 1: Khởi động Trace2Prompt

Bạn có thể chạy Trace2Prompt bằng 1 trong 2 cách sau:

**Cách 1: Build bằng Docker (Không cần cài Go)**
Nếu máy bạn có Docker, bạn có thể "mượn" Docker để biên dịch mã nguồn thành file chạy cục bộ một cách sạch sẽ:

```bash
git clone https://github.com/thuanDaoSE/trace2prompt.git
cd trace2prompt

# Với Mac/Linux:
docker run --rm -v $(pwd):/app -w /app golang:1.21 go build -o trace2prompt main.go otel_handlers.go prompt_generator.go mcp_server.go
./trace2prompt

# Với Windows (PowerShell):
docker run --rm -v ${PWD}:/app -w /app golang:1.21 go build -o trace2prompt.exe main.go otel_handlers.go prompt_generator.go mcp_server.go
.\trace2prompt.exe
```

**Cách 2: Tự Build bằng Go (Nếu máy đã cài Go)**

```bash
go build -o trace2prompt main.go otel_handlers.go prompt_generator.go mcp_server.go
./trace2prompt
```

_(Tool sẽ bắt đầu lắng nghe log ở cổng `localhost:4318` và mở giao diện Web tại `http://localhost:4319`)_

### Bước 2: Bật OTel cho dự án của bạn

> 💡 **Mẹo nhỏ (Pro Tip):** Lệnh khởi động khá dài, để tiện cho việc dev hàng ngày, anh em nên lưu lệnh này vào một file `run.bat` (với Windows) / `run.sh` (với Mac/Linux), hoặc đưa thẳng cấu hình các biến `-Dotel...` này vào file `launch.json` (VS Code) / Run Configuration (IntelliJ) nhé!

Đã test & hoạt động ổn định với OTel Agent v2.26.0.

Tải OpenTelemetry Java Agent v2.26.0:

```bash
curl -L -o opentelemetry-javaagent.jar "https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v2.26.0/opentelemetry-javaagent.jar"
```

Trace2Prompt sử dụng chuẩn OpenTelemetry (OTLP) quốc tế nên hỗ trợ **100% mọi ngôn ngữ lập trình**.

💡 **Lưu ý về Kiến trúc hệ thống:**

- **Dự án Monolith (Đơn khối):** Bạn chỉ cần thiết lập biến môi trường và gắn Agent vào dự án Backend duy nhất của bạn.
- **Dự án Microservices (Đa dịch vụ):** Tuyệt vời hơn nữa! Bạn chỉ cần lặp lại thao tác gắn Agent này cho **tất cả** các dịch vụ Backend của bạn (nhớ đổi tên `OTEL_SERVICE_NAME` cho từng cái). Trace2Prompt sẽ tự động nối vết (Distributed Tracing) các API gọi chéo nhau thành một luồng hoàn chỉnh!

👇 **Hãy click vào Stack của bạn bên dưới để xem hướng dẫn tích hợp:**

<details>
<summary><b>☕ Java (Spring Boot, Quarkus, v.v...)</b></summary>
<br>
**🪟 Dành cho Windows (Chạy trên 1 dòng lệnh):**

```bash
java -javaagent:opentelemetry-javaagent.jar "-Dotel.service.name=my-spring-app" "-Dotel.traces.exporter=otlp" "-Dotel.logs.exporter=otlp" "-Dotel.metrics.exporter=otlp" "-Dotel.exporter.otlp.endpoint=http://localhost:4318" "-Dotel.exporter.otlp.protocol=http/protobuf" "-Dotel.instrumentation.http.capture-headers.server.request=Authorization,Cookie,Accept,User-Agent,Content-Type" "-Dotel.instrumentation.http.server.capture-request-headers=Authorization,Cookie,Accept,User-Agent,Content-Type" "-Dotel.bsp.schedule.delay=500" "-Dotel.blrp.schedule.delay=500" -jar your-application.jar
```

**🐧 Dành cho Mac/Linux:**
Tải file `opentelemetry-javaagent.jar` và chạy lệnh sau:

```bash
java -javaagent:opentelemetry-javaagent.jar \
  -Dotel.service.name=my-spring-app \
  -Dotel.traces.exporter=otlp \
  -Dotel.logs.exporter=otlp \
  -Dotel.metrics.exporter=otlp \
  -Dotel.exporter.otlp.endpoint=http://localhost:4318 \
  -Dotel.exporter.otlp.protocol=http/protobuf \
  -Dotel.instrumentation.http.capture-headers.server.request=Authorization,Cookie,Accept,User-Agent,Content-Type \
  -Dotel.instrumentation.http.server.capture-request-headers=Authorization,Cookie,Accept,User-Agent,Content-Type \
  -Dotel.bsp.schedule.delay=500 \
  -jar your-application.jar
```

</details>

<details>
<summary><b>🟢 Node.js (Express, NestJS)</b></summary>
<br>

Cài đặt gói tự động (auto-instrumentation):

```bash
# Cài đặt thư viện cần thiết
npm install @opentelemetry/auto-instrumentations-node @opentelemetry/api
```

Sau đó khởi chạy ứng dụng (kèm theo Agent và Full cấu hình tối ưu):

```bash

# Chạy ứng dụng với các biến môi trường y hệt Java
env OTEL_SERVICE_NAME="node-backend-app" \
    OTEL_TRACES_EXPORTER="otlp" \
    OTEL_LOGS_EXPORTER="otlp" \
    OTEL_METRICS_EXPORTER="otlp" \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_INSTRUMENTATION_HTTP_CAPTURE_HEADERS_SERVER_REQUEST="Authorization,Cookie,Accept,User-Agent,Content-Type" \
    OTEL_BSP_SCHEDULE_DELAY=500 \
    OTEL_BLRP_SCHEDULE_DELAY=500 \
    node --require @opentelemetry/auto-instrumentations-node/register app.js
```

</details>

<details>
<summary><b>🐍 Python (Flask, Django, FastAPI)</b></summary>
<br>

Sử dụng bộ công cụ CLI của OpenTelemetry để tự động cài đặt các Sensor:

```bash
# Cài đặt công cụ bọc tự động của OTel
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install
```

Bọc lệnh chạy Python của bạn bằng lệnh `opentelemetry-instrument`:

```bash

# Khởi chạy ứng dụng với cấu hình chuẩn
env OTEL_SERVICE_NAME="python-backend-app" \
    OTEL_TRACES_EXPORTER="otlp" \
    OTEL_LOGS_EXPORTER="otlp" \
    OTEL_METRICS_EXPORTER="otlp" \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_INSTRUMENTATION_HTTP_CAPTURE_HEADERS_SERVER_REQUEST="Authorization,Cookie,Accept,User-Agent,Content-Type" \
    OTEL_BSP_SCHEDULE_DELAY=500 \
    OTEL_BLRP_SCHEDULE_DELAY=500 \
    opentelemetry-instrument python main.py
```

</details>

<details>
<summary><b>🐹 Golang (Gin, Fiber)</b></summary>
<br>

Với Go, bạn cần khởi tạo OTel Provider trong file `main.go`. Tham khảo [Tài liệu chính thức của OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/getting-started/). Sau khi cấu hình xong, chạy bình thường với các biến môi trường:

```bash
# Code Go sẽ tự động ăn theo các biến môi trường chuẩn của hệ điều hành
export OTEL_SERVICE_NAME="go-backend-app"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_BSP_SCHEDULE_DELAY=500
# ... sau đó chạy file thực thi
./my-go-app
```

</details>

<details>
<summary><b>⚛️ Frontend (React, Vue, Next.js, Vanilla JS)</b></summary>
<br>

Dù bạn dùng `axios`, `fetch`, hay `React Query`... Trace2Prompt đều tự động bắt trọn vẹn E2E Context nhờ cơ chế đánh chặn Native. Chỉ cần dán thẻ Script này vào thẻ `<head>` trong file `index.html` gốc của dự án:

```html
<script type="module" src="http://localhost:4319/trace2prompt.js"></script>
```

</details>

### Bước 3: Trải nghiệm ma thuật AI!

1. Tương tác với App của bạn và cố tình tạo ra một lỗi (Ví dụ: Thanh toán lỗi 500).
2. Mở trình duyệt vào `http://localhost:4319`.
3. Bấm **"Copy Prompt"** ở cái Trace báo đỏ chót.
4. Dán vào ChatGPT/Claude và để AI đọc trọn vẹn ngữ cảnh E2E rồi đưa ra code fix chính xác 100%!

---

## 🤖 Dành cho Autonomous AI Agents (MCP)

Dự án được tích hợp sẵn máy chủ **Model Context Protocol (MCP)** ở cổng `4318`.
Bạn có thể cấu hình IDE (như Cursor hoặc Claude Desktop) tự động gọi vào công cụ `get_latest_error_trace` của Trace2Prompt sau khi chạy test thất bại. Lúc này, AI sẽ tự động đọc E2E Trace, tự bắt bệnh và tự sửa code hoàn toàn tự động (Autonomous Agent Workflow)!

> ⚠️ **Lưu ý:** Tính năng Agentic Workflow này đang trong giai đoạn thử nghiệm (Beta), bạn có thể tự tùy chỉnh để MCP hoạt động với workflow của bạn theo ý muốn.

---

## 🤝 Đóng góp (Contributing)

Mã nguồn mở sống được là nhờ cộng đồng. Mọi ý tưởng tối ưu code, báo lỗi (Issue) hay Pull Request của các bạn đều được trân trọng!

## 📄 License

Dự án được phân phối dưới giấy phép MIT License.
