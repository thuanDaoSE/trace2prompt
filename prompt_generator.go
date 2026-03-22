package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Dictionary structure declaration
type PromptDict struct {
	Intro            string
	Env              string
	CpuMem           string
	WaitingOtel      string
	HttpCtx          string
	Frontend         string
	Browser          string
	BackendLog       string
	FlameGraph       string
	N1Warning        string
	WsAt             string
	Payload          string
	CurrentPage      string
	UnknownElem      string
	ClickAt          string
	FrontendCrash    string
	Error            string
	Trace            string
	FrontendApi      string
	Status           string
	Headers          string
	ReqBody          string
	ResBody          string
	NoFrontend       string
	NoLogs           string
	BackendExc       string
	NoException      string
	SystemErr        string
	SystemNote       string
	Truncated        string // Suffix appended when text is truncated
	TruncatedVI      string // Vietnamese truncation marker used in prettyJSON
	FrameworkHidden  string // Stacktrace filter label
	NoBackendWarning string // Warning when no backend trace found
	AuthAttached     string // Auth token present label
	AuthEmpty        string // Auth token absent label
	// Server-side console log messages
	StartupMsg           string // Printed on startup
	StartupHint          string // Printed as hint on startup
	MetricsUpdated       string // Printed when metrics are updated
	ReceivedSpan         string // Printed when a backend span arrives
	ReceivedFrontendSpan string // Printed when a frontend span arrives
	NewErrorCaught       string // Printed when a new E2E error is detected
	N1DetectedBuild      string // N+1 label inside buildTreeStr (not prompt)
	RepeatedCount        string
	NoErrorsYet          string
	WaitingForErrors     string
	IamAuth              string
	ExternalApi          string
	GraphqlOp            string
	RedisCmd             string
	MongoQuery           string
	SqlQuery             string
	InteractWith         string
	TopicQueue           string
	UnknownTopic         string
	MCPServerNotResp     string
	GlobalN1Title        string
	GlobalN1Warning      string
	GlobalN1Count        string
	JwtClaimsDecoded     string
	CookiesSent          string
	ExcMessage           string
	ExcStacktrace        string
}

// Create 2 languages
var dicts = map[string]PromptDict{
	"vi": {
		Intro:                "Vui lòng phân tích lỗi hệ thống dựa trên thông tin E2E Runtime Context dưới đây:\n\n",
		Env:                  "\n### 🖥️ MÔI TRƯỜNG & HẠ TẦNG\n",
		CpuMem:               "- 📊Tiêu thụ tài nguyên: `CPU %.1f%% | RAM %.0f MB`\n",
		WaitingOtel:          "- 📊 Tiêu thụ tài nguyên: `⏳ Đang chờ OTel thu thập...(cập nhật mỗi 60s)`\n",
		HttpCtx:              "\n### 🌐 HTTP REQUEST CONTEXT\n",
		Frontend:             "\n### 👣 HÀNH TRÌNH FRONTEND (USER JOURNEY)\n",
		Browser:              "  - 💻 Trình duyệt: `%s`\n",
		BackendLog:           "\n### 🛤️ HÀNH TRÌNH BACKEND (LOGS)\n",
		FlameGraph:           "\n### ⏳ THỨ TỰ THỰC THI & SQL (FLAME GRAPH)\n",
		N1Warning:            "%s⚠️ [N+1 DETECTED] ^ Lệnh DB trên bị vòng lặp gọi %d lần liên tục!\n",
		WsAt:                 "tại `%s`",
		Payload:              "  - Payload: `%s`\n",
		CurrentPage:          "Trang hiện tại",
		UnknownElem:          "Phần tử không xác định",
		ClickAt:              "tại `%s` (Phần tử: `%s`)",
		FrontendCrash:        "🛑 `FRONTEND CRASH`:\n  - Lỗi: `%s`\n",
		Error:                "  - Trace: `%s`\n",
		FrontendApi:          "🌐 `FRONTEND GỌI API` `%s %s` -> Trạng thái: `%s`",
		Status:               "  - 🎫 Headers: `%s`\n",
		Headers:              "  - 🎫 Headers: `%s`\n",
		ReqBody:              "  - 🔻 Request Body: %s\n",
		ResBody:              "  - 🔺 Response Body: %s\n",
		NoFrontend:           "- (Không ghi nhận được sự kiện Frontend nào. Vui lòng kiểm tra F12.)\n",
		NoLogs:               "- (Không có log info/warn nào được in ra trước khi lỗi)\n",
		BackendExc:           "\n### 🛑 BACKEND EXCEPTION STACKTRACE\n",
		NoException:          "- (Backend không văng Exception)\n",
		SystemErr:            "\n### ⚙️ SYSTEM BACKGROUND ERRORS (LỖI HỆ THỐNG GẦN ĐÂY)\n",
		SystemNote:           "> Chú ý: Các lỗi này sinh ra từ Background Threads (RabbitMQ, HikariCP, Memory...) và có thể là nguyên nhân gốc rễ gây ra lỗi hoặc ảnh hưởng tới Request trên.\n",
		Truncated:            "... [Đã cắt bớt do quá dài]",
		TruncatedVI:          "... [ĐÃ CẮT BỚT",
		FrameworkHidden:      "\t... [các lời gọi nội bộ framework đã ẩn]",
		NoBackendWarning:     "\n⚠️ [CẢNH BÁO] Không nhận được dữ liệu Flame Graph từ Backend (OTel Agent có thể bị trễ 5 giây, hoặc đây là request chỉ có Frontend).\n",
		AuthAttached:         "[TOKEN/COOKIE ĐÃ GỬI]",
		AuthEmpty:            "[TRỐNG - Không có Token/Cookie]",
		StartupMsg:           "🚀 Trace2Prompt (The AI Fixer) đang chạy nền trên cổng 4318...",
		StartupHint:          "👉 Cứ code bình thường. Khi có lỗi, tôi sẽ BÍP ngay và đẩy thẳng vào AI!",
		MetricsUpdated:       "📊 [Metrics] Đã cập nhật trạng thái RAM/CPU mới nhất!",
		ReceivedSpan:         "📥 Nhận Span bên dưới kết: %-15s | TraceID: %s",
		ReceivedFrontendSpan: "🔥 Nhận Span FRONTEND: %-15s | TraceID: %s",
		NewErrorCaught:       "🚨 Phát hiện lỗi E2E mới! (TraceID: %s)",
		N1DetectedBuild:      "%s⚠️ [N+1 DETECTED] ^ Truy vấn DB trên bị gọi trong vòng lặp %d lần!\n",
		RepeatedCount:        "- %s (⚠️ Lặp lại %d lần)\n",
		NoErrorsYet:          "Chưa có lỗi nào. Vui lòng test ứng dụng.",
		WaitingForErrors:     "⏳ Đang chờ hệ thống bắt lỗi... (Thử gọi API trả về lỗi 500)",
		IamAuth:              "🔐 [IAM AUTH]",
		ExternalApi:          "🌍 [EXTERNAL API]",
		GraphqlOp:            "Operation:",
		RedisCmd:             "Command:",
		MongoQuery:           "Query:",
		SqlQuery:             "Query:",
		InteractWith:         "GIAO TIẾP VỚI",
		TopicQueue:           "🎯 Topic/Queue:",
		UnknownTopic:         "Topic không xác định",
		MCPServerNotResp:     "⚠️ Trace2Prompt Server không phản hồi. Vui lòng kiểm tra xem Trace2Prompt.exe có đang chạy không, hoặc nhấn Enter trên màn hình đen nếu nó bị treo.",
		GlobalN1Title:        "\n### 🚨 GLOBAL N+1 QUERY DETECTED (PHÁT HIỆN N+1 TỔNG THỂ)\n",
		GlobalN1Warning:      "> ⚠️ Hệ thống phát hiện các câu SQL dưới đây bị gọi lặp lại rải rác rất nhiều lần. Vui lòng kiểm tra vòng lặp Code hoặc cấu hình FetchType/EntityGraph.\n\n",
		GlobalN1Count:        "- 🔁 Bị gọi **%d lần**:\n```sql\n%s\n```\n",
		JwtClaimsDecoded:     "\n- **JWT Claims (Tự động giải mã):** \n```json\n%s\n```",
		CookiesSent:          "\n- **Cookies đã gửi (Chỉ hiện Keys):** `[%s]`",
		ExcMessage:           "**Thông báo lỗi:** %s\n",
		ExcStacktrace:        "**Stacktrace:**\n```text\n%s\n```\n",
	},
	"en": {
		Intro:                "Please analyze the system error based on the E2E Runtime Context below:\n\n",
		Env:                  "\n### 🖥️ ENVIRONMENT & INFRASTRUCTURE\n",
		CpuMem:               "- 📊 Process Metrics: `CPU %.1f%% | RAM %.0f MB`\n",
		WaitingOtel:          "- 📊 Process Metrics: `⏳ Waiting for OTel agent...(update per 60s)`\n",
		HttpCtx:              "\n### 🌐 HTTP REQUEST CONTEXT\n",
		Frontend:             "\n### 👣 FRONTEND JOURNEY (USER JOURNEY)\n",
		Browser:              "  - 💻 Browser: `%s`\n",
		BackendLog:           "\n### 🛤️ BACKEND JOURNEY (LOGS)\n",
		FlameGraph:           "\n### ⏳ EXECUTION ORDER & SQL (FLAME GRAPH)\n",
		N1Warning:            "%s⚠️ [N+1 DETECTED] ^ The above DB query is called in a loop %d times!\n",
		WsAt:                 "at `%s`",
		Payload:              "  - Payload: `%s`\n",
		CurrentPage:          "Current page",
		UnknownElem:          "Unknown Element",
		ClickAt:              "at `%s` (Element: `%s`)",
		FrontendCrash:        "🛑 `FRONTEND CRASH`:\n  - Error: `%s`\n",
		Error:                "  - Trace: `%s`\n",
		FrontendApi:          "🌐 `FRONTEND API CALL` `%s %s` -> Status: `%s`",
		Status:               "  - 🎫 Headers: `%s`\n",
		Headers:              "  - 🎫 Headers: `%s`\n",
		ReqBody:              "  - 🔻 Request Body: %s\n",
		ResBody:              "  - 🔺 Response Body: %s\n",
		NoFrontend:           "- (No Frontend events recorded. Please check F12.)\n",
		NoLogs:               "- (No info/warn logs printed before error)\n",
		BackendExc:           "\n### 🛑 BACKEND EXCEPTION STACKTRACE\n",
		NoException:          "- (Backend did not throw Exception)\n",
		SystemErr:            "\n### ⚙️ SYSTEM BACKGROUND ERRORS (RECENT SYSTEM ERRORS)\n",
		SystemNote:           "> Note: These errors are generated from Background Threads (RabbitMQ, HikariCP, Memory...) and may be the root cause of the error or affect the above Request.\n",
		Truncated:            "... [Truncated due to excessive length]",
		TruncatedVI:          "... [Truncated due to excessive length]",
		FrameworkHidden:      "\t... [framework internal calls hidden]",
		NoBackendWarning:     "\n⚠️ [WARNING] No Flame Graph data received from Backend (OTel Agent may be delayed 5s, or this is a Frontend-only request).\n",
		AuthAttached:         "[TOKEN/COOKIE ATTACHED]",
		AuthEmpty:            "[EMPTY - No Token/Cookie received]",
		StartupMsg:           "🚀 Trace2Prompt (The AI Fixer) is running in background on port 4318...",
		StartupHint:          "👉 Just code normally. When there are errors, I will BEEP and serve it directly to AI!",
		MetricsUpdated:       "📊 [Metrics] Updated latest RAM/CPU status!",
		ReceivedSpan:         "📥 Received Span: %-15s | TraceID: %s",
		ReceivedFrontendSpan: "🔥 Received FRONTEND Span: %-15s | TraceID: %s",
		NewErrorCaught:       "🚨 CAUGHT NEW E2E ERROR! (TraceID: %s)",
		N1DetectedBuild:      "%s⚠️ [N+1 DETECTED] ^ The above DB query is called in a loop %d times!\n",
		RepeatedCount:        "- %s (⚠️ Repeated %d times)\n",
		NoErrorsYet:          "No errors yet. Please test the application.",
		WaitingForErrors:     "⏳ Waiting for system to catch errors... (Try calling API that throws 500 error)",
		IamAuth:              "🔐 [IAM AUTH]",
		ExternalApi:          "🌍 [EXTERNAL API]",
		GraphqlOp:            "Operation:",
		RedisCmd:             "Command:",
		MongoQuery:           "Query:",
		SqlQuery:             "Query:",
		InteractWith:         "INTERACT WITH",
		TopicQueue:           "🎯 Topic/Queue:",
		UnknownTopic:         "Unknown Topic",
		MCPServerNotResp:     "⚠️ Trace2Prompt Server not responding. Please check if you have enabled Trace2Prompt.exe running in background, or press Enter on the black screen if it's frozen.",
		GlobalN1Title:        "\n### 🚨 GLOBAL N+1 QUERY DETECTED\n",
		GlobalN1Warning:      "> ⚠️ The system detected that the following SQL queries are repeatedly called many times. Please check your code loops or FetchType/EntityGraph configurations.\n\n",
		GlobalN1Count:        "- 🔁 Called **%d times**:\n```sql\n%s\n```\n",
		JwtClaimsDecoded:     "\n- **JWT Claims (Auto decoded):** \n```json\n%s\n```",
		CookiesSent:          "\n- **Cookies sent (Only show Keys):** `[%s]`",
		ExcMessage:           "**Message:** %s\n",
		ExcStacktrace:        "**Stacktrace:**\n```text\n%s\n```\n",
	},
}

// serverLang controls which language is used for server-side console log messages.
// Set via --lang CLI flag in main.go (default: "en").
var serverLang = "en"

// getServerDict returns the PromptDict for the current server language.
func getServerDict() PromptDict {
	if d, ok := dicts[serverLang]; ok {
		return d
	}
	return dicts["en"]
}

func maskSensitiveData(text string) string {
	if text == "" {
		return text
	}

	for _, rule := range CompiledMaskingRules {
		text = rule.Regex.ReplaceAllString(text, rule.Replace)
	}

	return text
}

// JSON beautifier function for readers
// JSON beautifier function for readers (UPGRADED TO HANDLE TRUNCATED JSON)
func prettyJSONWithMarker(raw string, truncatedMarker string) string {
	if raw == "" {
		return ""
	}

	// 1. If raw contains the truncation marker, temporarily remove it to process JSON
	suffix := ""
	cleanRaw := raw
	if idx := strings.Index(raw, truncatedMarker); idx != -1 {
		cleanRaw = raw[:idx]
		suffix = "\n    " + raw[idx:]
	}

	// 2. Try to format original JSON first
	var prettyBuf bytes.Buffer
	err := json.Indent(&prettyBuf, []byte(cleanRaw), "    ", "  ")

	// 3. IF ERROR (DUE TO TRUNCATED JSON FROM FRONTEND) -> USE "BRACKET COUNTING" ALGORITHM TO PATCH
	if err != nil {
		// Temporarily patch by counting opening brackets {, [ and automatically closing }, ]
		openBraces := strings.Count(cleanRaw, "{") - strings.Count(cleanRaw, "}")
		openBrackets := strings.Count(cleanRaw, "[") - strings.Count(cleanRaw, "]")

		patchedRaw := cleanRaw
		// Fix truncated string (example: "quan )
		if strings.Count(patchedRaw, "\"")%2 != 0 {
			patchedRaw += "\""
		}

		for i := 0; i < openBraces; i++ {
			patchedRaw += "}"
		}
		for i := 0; i < openBrackets; i++ {
			patchedRaw += "]"
		}

		// Try to format JSON again after patching
		var patchedBuf bytes.Buffer
		err2 := json.Indent(&patchedBuf, []byte(patchedRaw), "    ", "  ")
		if err2 == nil {
			return "\n    " + patchedBuf.String() + suffix
		}

		// If still heavily error-prone, manually break lines for better readability
		manualFormat := strings.ReplaceAll(cleanRaw, "\",\"", "\",\n      \"")
		manualFormat = strings.ReplaceAll(manualFormat, "},{", "},\n    {")
		return "\n    " + manualFormat + suffix
	}

	return "\n    " + prettyBuf.String() + suffix
}

func prettyJSON(raw string) string {
	// Default: use the Vietnamese marker (backward compat — same as before)
	return prettyJSONWithMarker(raw, "... [Truncated due to excessive length]")
}

func prettyJSONL(raw string, t PromptDict) string {
	return prettyJSONWithMarker(raw, t.TruncatedVI)
}

// truncateText is a global version (lang-agnostic) used in non-prompt contexts.
// For prompt output use truncateTextL (language-aware).
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "... [Truncated due to excessive length]"
}

func truncateTextL(text string, maxLen int, t PromptDict) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + t.Truncated
}

func filterStacktrace(rawStacktrace string) string {
	return filterStacktraceL(rawStacktrace, dicts["en"])
}

func filterStacktraceL(rawStacktrace string, t PromptDict) string {
	lines := strings.Split(strings.TrimSpace(rawStacktrace), "\n")
	if len(lines) == 0 {
		return rawStacktrace
	}

	var result []string
	result = append(result, lines[0])

	skippedCount := 0
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if i <= 3 || strings.Contains(trimmedLine, ProjectNamespace) {
			result = append(result, line)
			skippedCount = 0
		} else {
			if skippedCount == 0 {
				result = append(result, t.FrameworkHidden)
			}
			skippedCount++
		}
	}
	return strings.Join(result, "\n")
}

// =====================================================================
// MAIN FUNCTION HAS BEEN EXTREMELY CLEANLY REFACTORED
// =====================================================================
func generateE2EPrompt(targetTraceID string, lang string) string {
	t, ok := dicts[lang]
	if !ok {
		t = dicts["en"]
	}

	// Lock Mutex once in main function for safety for all child functions accessing data
	mu.Lock()
	defer mu.Unlock()

	// 1. Find Backend Trace
	bestTrace, hasBackend := findBestTraceMatch(targetTraceID)

	var sb strings.Builder
	sb.WriteString(t.Intro)
	sb.WriteString("=================================================\n")
	sb.WriteString(fmt.Sprintf("TraceID: `%s`\n", targetTraceID))

	// 2. Print Environment & Infrastructure
	buildEnvironmentContext(&sb, bestTrace, hasBackend, t)

	// 3. Print Frontend Journey
	buildFrontendJourney(&sb, targetTraceID, bestTrace, hasBackend, t)

	// 4. Print Backend Journey (Logs, Exception, Flame Graph)
	if hasBackend {
		buildBackendJourney(&sb, bestTrace, t)
	}

	// 5. Print System Errors
	buildSystemErrors(&sb, t)

	sb.WriteString("\n=================================================\n")
	return sb.String()
}

// =====================================================================
// HELPER FUNCTIONS BELOW
// =====================================================================

// 1. Logic to find Trace (Fuzzy Match)
func findBestTraceMatch(targetTraceID string) (*TraceRecord, bool) {
	bestTrace, exists := traceMap[targetTraceID]

	// REVERSE FUZZY MATCH: Get URL & Time info from Frontend to find reverse in Backend
	if !exists {
		var targetUrl string
		var targetTime time.Time
		for i := len(spanBuffer) - 1; i >= 0; i-- {
			s := spanBuffer[i]
			if s.TraceID == targetTraceID && s.Name != "click" && s.Name != "EXCEPTION" {
				targetUrl = s.Attributes["http.url"]
				targetTime = s.Timestamp
				break
			}
		}

		if targetUrl != "" {
			for _, tr := range traceMap {
				if tr.HttpUrl != "" && strings.Contains(targetUrl, tr.HttpUrl) {
					diff := tr.LastUpdated.Sub(targetTime)
					if diff < 0 {
						diff = -diff
					}

					if diff <= 3*time.Second {
						bestTrace = tr
						exists = true
						break
					}
				}
			}
		}
	}
	return bestTrace, exists
}

// 2. Logic to print Environment info (CPU, RAM, OS, DB, HTTP)
func buildEnvironmentContext(sb *strings.Builder, bestTrace *TraceRecord, hasBackend bool, t PromptDict) {
	if !hasBackend {
		sb.WriteString(t.NoBackendWarning)
		return
	}

	bestTrace.Printed = true
	sb.WriteString(t.Env)
	sb.WriteString(fmt.Sprintf("- Service: `%s`\n", bestTrace.ServiceName))
	sb.WriteString(fmt.Sprintf("- OS: `%s`\n", bestTrace.OsDesc))
	sb.WriteString(fmt.Sprintf("- Runtime: `%s`\n", bestTrace.JavaVersion))
	sb.WriteString(fmt.Sprintf("- Database: `%s @ %s`\n", bestTrace.DbSystem, bestTrace.DbAddress))

	if bestTrace.SnapshotRAM > 0 || bestTrace.SnapshotCPU > 0 {
		sb.WriteString(fmt.Sprintf(t.CpuMem, bestTrace.SnapshotCPU, bestTrace.SnapshotRAM))
	} else {
		sb.WriteString(t.WaitingOtel)
	}

	if bestTrace.K8sPodName != "" {
		sb.WriteString(fmt.Sprintf("- ☸️ K8s Pod: `%s`\n", bestTrace.K8sPodName))
	}
	if bestTrace.ContainerID != "" {
		shortId := bestTrace.ContainerID
		if len(shortId) > 12 {
			shortId = shortId[:12]
		}
		sb.WriteString(fmt.Sprintf("- 🐳 Docker Container: `%s`\n", shortId))
	}

	sb.WriteString(t.HttpCtx)
	sb.WriteString(fmt.Sprintf("- Method: `%s`\n", bestTrace.HttpMethod))
	sb.WriteString(fmt.Sprintf("- URL: `%s`\n", bestTrace.HttpUrl))
	sb.WriteString(fmt.Sprintf("- Status Code: `%s`\n", bestTrace.HttpStatus))

	if bestTrace.AuthToken != "" {
		sb.WriteString(fmt.Sprintf("- 🔐 Backend Received Auth: `%s`\n", t.AuthAttached))
		if bestTrace.AuthContext != "" {
			sb.WriteString(bestTrace.AuthContext + "\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("- 🔓 Backend Received Auth: `%s`\n", t.AuthEmpty))
	}
}

// 3. Logic to print Frontend journey (Clicks, API, WebSocket, Console, Errors)
func buildFrontendJourney(sb *strings.Builder, targetTraceID string, bestTrace *TraceRecord, hasBackend bool, t PromptDict) {
	sb.WriteString(t.Frontend)
	foundFrontend := false
	apiPrinted := false

	lastWsUrl := ""
	lastWsType := ""
	wsSpamCount := 0

	// 🌟 USE TRACE TIME AS BASELINE
	var targetTime time.Time
	for _, s := range spanBuffer {
		if s.TraceID == targetTraceID {
			targetTime = s.Timestamp
			break
		}
	}

	for i := len(spanBuffer) - 1; i >= 0; i-- {
		s := spanBuffer[i]
		timeStr := s.Timestamp.Format("15:04:05")

		// 🌟 UPGRADED MAIN LOGIC: FUZZY MATCH FOR THE ENTIRE FRONTEND
		// If same TraceID -> Definitely include
		// If different TraceID, but occurs within 10 seconds before Backend error -> Also include!
		isValidTimeContext := false
		if !targetTime.IsZero() {
			timeDiff := targetTime.Sub(s.Timestamp)
			if timeDiff >= -2*time.Second && timeDiff <= 10*time.Second {
				isValidTimeContext = true
			}
		}

		if s.TraceID != targetTraceID && !isValidTimeContext {
			continue // Skip irrelevant or old events
		}

		// 1. CATCH WEBSOCKET EVENTS
		if strings.HasPrefix(s.Name, "WS ") {
			url := s.Attributes["http.url"]
			if url == lastWsUrl && s.Name == lastWsType {
				wsSpamCount++
				continue
			}
			if wsSpamCount > 0 {
				sb.WriteString(fmt.Sprintf("- 🔄 (skip %d duplicate %s messages)\n", wsSpamCount, lastWsType))
				wsSpamCount = 0
			}
			payload := s.Attributes["messaging.payload"]
			icon := "📤"
			if s.Name == "WS RECEIVE" {
				icon = "📥"
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s `%s` %s\n", timeStr, icon, s.Name, fmt.Sprintf(t.WsAt, url)))
			if payload != "" {
				sb.WriteString(fmt.Sprintf(t.Payload, payload))
			}
			foundFrontend = true
			lastWsUrl = url
			lastWsType = s.Name
			continue
		}

		// 2. CLICK EVENTS
		if s.Name == "click" || strings.HasPrefix(s.Name, "Navigation") {
			xpath := s.Attributes["target_xpath"]
			if xpath == "" {
				xpath = s.Attributes["target_id"]
			}
			url := s.Attributes["http.url"]
			if url == "" {
				url = t.CurrentPage
			}
			if xpath == "" {
				xpath = t.UnknownElem
			}
			sb.WriteString(fmt.Sprintf("- [%s] 🖱️ `CLICK` %s\n", timeStr, fmt.Sprintf(t.ClickAt, url, xpath)))
			foundFrontend = true
		}

		// 3. FRONTEND CONSOLE LOGS & WARNINGS
		if strings.Contains(s.Name, "CONSOLE_") {
			msg := s.Attributes["console.message"]
			if msg != "" {
				// Truncate long logs to avoid spam
				sb.WriteString(fmt.Sprintf("- [%s] %s: `%s`\n", timeStr, s.Name, truncateTextL(msg, 200, t)))
				foundFrontend = true
			}
		}

		// 4. FRONTEND JS CRASH & PROMISE REJECT
		if strings.Contains(s.Name, "CRASH") || strings.Contains(s.Name, "REJECT") || strings.Contains(s.Name, "RESOURCE_ERROR") || s.Name == "EXCEPTION" {
			errMsg := s.Attributes["error.message"]
			if errMsg == "" {
				errMsg = s.Attributes["exception.message"]
			}
			errStack := s.Attributes["error.stack"]
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", timeStr, fmt.Sprintf(t.FrontendCrash, errMsg)))
			if errStack != "" {
				sb.WriteString(fmt.Sprintf(t.Error, truncateTextL(errStack, 300, t)))
			}
			foundFrontend = true
		}

		// 5. FRONTEND API CALL
		if !apiPrinted && !strings.Contains(s.Name, "CONSOLE_") && !strings.Contains(s.Name, "CRASH") && !strings.Contains(s.Name, "REJECT") && s.Name != "click" && s.Name != "EXCEPTION" && s.Name != "🖼️ RESOURCE_ERROR" {
			url := s.Attributes["http.url"]
			method := s.Attributes["http.request.method"]
			if method == "" {
				method = s.Attributes["http.method"]
			}

			if url != "" && method != "" {
				isSameTrace := s.TraceID == targetTraceID
				isFuzzyMatch := hasBackend && bestTrace.HttpUrl != "" && strings.Contains(url, bestTrace.HttpUrl)

				// ONLY PRINT IF IT EXACTLY MATCHES THE FAILING API IN THE BACKEND
				if isSameTrace || isFuzzyMatch {
					status := s.Attributes["http.response.status_code"]
					if status == "" {
						status = s.Attributes["http.status_code"]
					}
					reqBody := s.Attributes["http.request.body"]
					resBody := s.Attributes["http.response.body"]
					headers := s.Attributes["http.request.headers"]
					userAgent := s.Attributes["http.user_agent"]
					reqSize := s.Attributes["http.request.size"]
					resSize := s.Attributes["http.response.size"]
					currentUrl := s.Attributes["page.current_url"]
					networkStat := s.Attributes["network.status"]
					viewport := s.Attributes["device.viewport"]

					sb.WriteString(fmt.Sprintf("- [%s] %s\n", timeStr, fmt.Sprintf(t.FrontendApi, method, url, status)))
					if currentUrl != "" {
						sb.WriteString(fmt.Sprintf("  - %s: `%s`\n", t.CurrentPage, currentUrl))
						sb.WriteString(fmt.Sprintf("  - Network: `%s` | Screen: `%s`\n", networkStat, viewport))
					}
					if userAgent != "" {
						sb.WriteString(fmt.Sprintf(t.Browser, userAgent))
					}
					if headers != "" && headers != "{}" {
						sb.WriteString(fmt.Sprintf(t.Headers, truncateTextL(headers, 200, t)))
					}
					if reqBody != "" {
						sizeInfo := ""
						if reqSize != "" {
							sizeInfo = fmt.Sprintf(" (Size: %s)", reqSize)
						}
						sb.WriteString(fmt.Sprintf("  - 🔻 Request Body%s: %s\n", sizeInfo, prettyJSONL(maskSensitiveData(reqBody), t)))
					}
					if resBody != "" {
						sizeInfo := ""
						if resSize != "" {
							sizeInfo = fmt.Sprintf(" (Size: %s)", resSize)
						}
						sb.WriteString(fmt.Sprintf("  - 🔺 Response Body%s: %s\n", sizeInfo, prettyJSONL(maskSensitiveData(resBody), t)))
					}
					foundFrontend = true
					apiPrinted = true
				}
			}
		}
	}

	if wsSpamCount > 0 {
		sb.WriteString(fmt.Sprintf("- 🔄 (Skip %d duplicate %s messages)\n", wsSpamCount, lastWsType))
	}
	if !foundFrontend {
		sb.WriteString(t.NoFrontend)
	}
}

// 4. Logic to print Backend Logs, Exceptions & Flame Graph
func buildBackendJourney(sb *strings.Builder, bestTrace *TraceRecord, t PromptDict) {
	// BACKEND LOGS
	sb.WriteString(t.BackendLog)
	for _, logMsg := range bestTrace.Logs {
		sb.WriteString("- " + maskSensitiveData(logMsg) + "\n")
	}
	if len(bestTrace.Logs) == 0 {
		sb.WriteString(t.NoLogs)
	}

	// BACKEND EXCEPTION
	sb.WriteString(t.BackendExc)
	uniqueErrors := make(map[string]bool)
	for _, errMsg := range bestTrace.ErrorMsgs {
		if !uniqueErrors[errMsg] {
			sb.WriteString(errMsg + "\n")
			uniqueErrors[errMsg] = true
		}
	}
	if len(bestTrace.ErrorMsgs) == 0 {
		sb.WriteString(t.NoException)
	}

	// FLAME GRAPH
	sb.WriteString(t.FlameGraph)
	for _, span := range bestTrace.Spans {
		span.Children = nil
	}

	var roots []*SpanNode
	for _, span := range bestTrace.Spans {
		if span.ParentSpanID == "" {
			roots = append(roots, span)
		} else {
			if parent, ok := bestTrace.Spans[span.ParentSpanID]; ok {
				parent.Children = append(parent.Children, span)
			} else {
				roots = append(roots, span)
			}
		}
	}
	for _, root := range roots {
		buildTreeStr(root, 0, sb)
	}

	// ==========================================
	// 🚨 GLOBAL N+1 SCANNER
	// ==========================================
	sqlCounter := make(map[string]int)
	for _, span := range bestTrace.Spans {
		if span.SQL != "" {
			sqlCounter[span.SQL]++
		}
	}

	hasGlobalN1 := false
	for sql, count := range sqlCounter {
		if count >= 3 { // If the SQL query appears 3 or more times in 1 Request
			if !hasGlobalN1 {
				sb.WriteString(t.GlobalN1Title)
				sb.WriteString(t.GlobalN1Warning)
				hasGlobalN1 = true
			}
			sb.WriteString(fmt.Sprintf(t.GlobalN1Count, count, formatSQL(sql)))
		}
	}
}

// 5. Logic to print System Background Errors
func buildSystemErrors(sb *strings.Builder, t PromptDict) {
	if len(systemLogsBuffer) == 0 {
		return
	}
	sb.WriteString(t.SystemErr)
	sb.WriteString(t.SystemNote)
	errorCount := make(map[string]int)
	var errorOrder []string
	var latestFullMsg = make(map[string]string)

	for _, sysErr := range systemLogsBuffer {
		sig := sysErr

		if len(sysErr) > 11 && strings.HasPrefix(sysErr, "[") && sysErr[9] == ']' {
			sig = sysErr[11:]
		}

		if errorCount[sig] == 0 {

			errorOrder = append(errorOrder, sig)

		}

		errorCount[sig]++
		latestFullMsg[sig] = sysErr

	}

	startIndex := 0
	if len(errorOrder) > 5 {
		startIndex = len(errorOrder) - 5
	}

	for i := startIndex; i < len(errorOrder); i++ {
		sig := errorOrder[i]
		count := errorCount[sig]
		fullMsg := latestFullMsg[sig]

		if count > 1 {
			sb.WriteString(fmt.Sprintf(t.RepeatedCount, fullMsg, count))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", fullMsg))
		}
	}
}
