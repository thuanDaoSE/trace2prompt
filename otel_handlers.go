package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"bytes"             
	"encoding/base64"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)
// 1. Function to only get Cookie Names, discard Values (To avoid leaks)
func extractCookieKeys(cookieHeader string) string {
	if cookieHeader == "" { return "" }
	parts := strings.Split(cookieHeader, ";")
	var keys []string
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) > 0 && kv[0] != "" {
			keys = append(keys, kv[0])
		}
	}
	return strings.Join(keys, ", ")
}
const MAX_RECENT_TRACES = 100

// 2. Function to AUTO DECODE JWT PAYLOAD (Extremely useful for AI)
func extractJwtPayload(authHeader string) string {
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	token := strings.TrimSpace(authHeader[7:])
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return "" } // Not standard JWT

	// Decode the second part (Payload)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return "" }
    
	// Reformat JSON nicely for Prompt output
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, payload, "", "  "); err == nil {
		return prettyJSON.String()
	}
	return string(payload)
}


func startServers() {

http.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
		// 🌟 SECTION 2: ADD CORS TO ALLOW BROWSER FRONTEND TO SEND LOGS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		body, _ := io.ReadAll(r.Body)
		req := ptraceotlp.NewExportRequest()
		req.UnmarshalProto(body)

		mu.Lock()
		for i := 0; i < req.Traces().ResourceSpans().Len(); i++ {
			rs := req.Traces().ResourceSpans().At(i)
			resAttrs := rs.Resource().Attributes()
			osDesc, javaVer, svcName := "", "", ""
			
			if val, ok := resAttrs.Get("os.description"); ok { osDesc = val.AsString() }
			if val, ok := resAttrs.Get("process.runtime.version"); ok { javaVer = val.AsString() }
			if val, ok := resAttrs.Get("service.name"); ok { svcName = val.AsString() }

			k8sPod, containerId := "", ""
			if val, ok := resAttrs.Get("k8s.pod.name"); ok { k8sPod = val.AsString() }
			if val, ok := resAttrs.Get("container.id"); ok { containerId = val.AsString() }

			for j := 0; j < rs.ScopeSpans().Len(); j++ {
				ss := rs.ScopeSpans().At(j)
				for k := 0; k < ss.Spans().Len(); k++ {
					span := ss.Spans().At(k)
					tID := span.TraceID().String()
					if tID == "" { continue }

					// Save to ring buffer
					spanRec := SpanRecord{
						TraceID:    tID,
						SpanID:     span.SpanID().String(),
						ParentSpanID: span.ParentSpanID().String(),
						Name:       span.Name(),
						Timestamp:  span.StartTimestamp().AsTime().Local(), 
						Attributes: make(map[string]string),
						HasError:   span.Status().Code() == 2,
					}

					span.Attributes().Range(func(k string, v pcommon.Value) bool {
						valStr := v.AsString()
						
						// Mask sensitive data
						lowerKey := strings.ToLower(k)
						if strings.Contains(lowerKey, "password") || 
						   strings.Contains(lowerKey, "secret") || 
						   strings.Contains(lowerKey, "token") || 
						   strings.Contains(lowerKey, "api_key") || 
						   strings.Contains(lowerKey, "apikey") || 
						   strings.Contains(lowerKey, "credential") ||
						   strings.Contains(lowerKey, "cookie") {
							
							spanRec.Attributes[k] = "[REDACTED_SECRET]"
							
						} else {
							// If normal field, still scan through maskText function (hide email/bearer token in long text)
							spanRec.Attributes[k] = maskSensitiveData(valStr) 
						}
						return true
					})
					fmt.Printf(getServerDict().ReceivedSpan+"\n", span.Name(), tID)
					// Push to array of 200 events
					if len(spanBuffer) >= MaxBufferSize {
						spanBuffer = spanBuffer[1:]
					}
					spanBuffer = append(spanBuffer, spanRec)

					// Trigger if Frontend reports HTTP 5xx error
					if spanRec.HasError && spanRec.Attributes["http.status_code"] != "" {
						triggerE2EPromptCreation(tID)
					}

					// Old logic to save to traceMap for drawing Backend Flamegraph
					tr := getOrCreateTrace(tID)

					// Camera snapshot metrics for every request (both good and bad)
					if tr.SnapshotRAM == 0 {
						latestMetrics.mu.Lock()
						tr.SnapshotCPU = latestMetrics.CPUUsage
						tr.SnapshotRAM = latestMetrics.RAMUsed
						latestMetrics.mu.Unlock()
					}

					if osDesc != "" { tr.OsDesc = osDesc }
					if javaVer != "" { tr.JavaVersion = javaVer }
					if svcName != "" { tr.ServiceName = svcName }
					if k8sPod != "" { tr.K8sPodName = k8sPod }          // Attach it
					if containerId != "" { tr.ContainerID = containerId } // Attach it

					dur := (span.EndTimestamp() - span.StartTimestamp()) / 1000000
					sql, msgSys, msgOp, msgDest, spanDbSys := "", "", "", "", "" 
					extUrl, extMethod := "", ""

					attrs := span.Attributes()
					if sqlVal, ok := attrs.Get("db.statement"); ok { sql = sqlVal.Str() }
					if dbVal, ok := attrs.Get("db.system"); ok { spanDbSys = dbVal.AsString() }
					
					rpcSys, faasTrig, gqlOp := "", "", ""
					if val, ok := attrs.Get("rpc.system"); ok { rpcSys = val.AsString() }
					if val, ok := attrs.Get("faas.trigger"); ok { faasTrig = val.AsString() }
					if val, ok := attrs.Get("graphql.operation.name"); ok { gqlOp = val.AsString() }

					if val, ok := attrs.Get("graphql.document"); ok { sql = val.AsString() }

					if msgVal, ok := attrs.Get("messaging.system"); ok { msgSys = msgVal.AsString() }
					if val, ok := attrs.Get("messaging.operation"); ok { msgOp = val.AsString() }
					if val, ok := attrs.Get("messaging.destination.name"); ok { msgDest = val.AsString() }


					// Catch external API calls
					if val, ok := attrs.Get("http.url"); ok { extUrl = val.AsString() }
					if val, ok := attrs.Get("url.full"); ok { extUrl = val.AsString() } // New OTel often uses url.full
					if val, ok := attrs.Get("http.method"); ok { extMethod = val.AsString() }
					if val, ok := attrs.Get("http.request.method"); ok { extMethod = val.AsString() }


					if val, ok := attrs.Get("db.system"); ok { tr.DbSystem = val.AsString() }
					if val, ok := attrs.Get("server.address"); ok {
						port := ""
						if pVal, pok := attrs.Get("server.port"); pok { port = ":" + pVal.AsString() }
						tr.DbAddress = maskSensitiveData(val.AsString() + port)
					}
					if val, ok := attrs.Get("http.request.method"); ok { tr.HttpMethod = val.AsString() }
					if val, ok := attrs.Get("http.method"); ok { tr.HttpMethod = val.AsString() }
					if val, ok := attrs.Get("http.target"); ok { tr.HttpUrl = val.AsString() }
					if val, ok := attrs.Get("url.path"); ok { 
						tr.HttpUrl = val.AsString() 
						if qVal, qOk := attrs.Get("url.query"); qOk { tr.HttpUrl += "?" + qVal.AsString() }
					}
					if val, ok := attrs.Get("http.response.status_code"); ok { tr.HttpStatus = val.AsString() }
					if val, ok := attrs.Get("http.request.header.user-agent"); ok { tr.UserAgent = val.AsString() }
					// Smart authentication scanning
					
					var cookieKeys, jwtClaims string

					
					
					attrs.Range(func(k string, v pcommon.Value) bool {
						lowerK := strings.ToLower(k)
						valStr := v.AsString()

						// Auto analyze Authorization Header
						if strings.Contains(lowerK, "authorization") {
							jwtClaims = extractJwtPayload(valStr)
							if !strings.Contains(tr.AuthToken, "[Authorization]") {
								tr.AuthToken += "[Authorization] "
							}
						}
						// Auto extract Cookie Names
						if strings.Contains(lowerK, "cookie") {
							keys := extractCookieKeys(valStr)
							if keys != "" {
								cookieKeys = keys
							}
							if !strings.Contains(tr.AuthToken, "[Cookie]") {
								tr.AuthToken += "[Cookie] "
							}
						}
						return true
					})

					// Save this super context for prompt_generator.go
					if jwtClaims != "" {
						tr.AuthContext = "\n- **JWT Claims (Auto decoded):** \n```json\n" + jwtClaims + "\n```"
					} else if cookieKeys != "" {
						tr.AuthContext = "\n- **Cookies sent (Only show Keys):** `[" + cookieKeys + "]`"
					}
					tr.Spans[span.SpanID().String()] = &SpanNode{
						SpanID:       span.SpanID().String(),
						ParentSpanID: span.ParentSpanID().String(),
						Name:         span.Name(),
						DurationMs:   int64(dur),
						SQL:          sql,
						MsgSystem:    msgSys,
						DbSystem:     spanDbSys,
						ExtHttpUrl:    extUrl,   
						ExtHttpMethod: extMethod,
						ServiceName:  svcName,	
						MsgOperation: strings.ToUpper(msgOp), // PUBLISH / RECEIVE
						MsgDestName:  msgDest,
						RpcSystem:        rpcSys,
						FaasTrigger:      faasTrig,
						GraphqlOperation: gqlOp,
					}
				}
			}
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := plogotlp.NewExportRequest()
		req.UnmarshalProto(body)

		for i := 0; i < req.Logs().ResourceLogs().Len(); i++ {
			rl := req.Logs().ResourceLogs().At(i)
			for j := 0; j < rl.ScopeLogs().Len(); j++ {
				sl := rl.ScopeLogs().At(j)
				
				// Get Class name that generated the Log
				scopeName := sl.Scope().Name()
				parts := strings.Split(scopeName, ".")
				shortScope := parts[len(parts)-1]

				for k := 0; k < sl.LogRecords().Len(); k++ {
					lr := sl.LogRecords().At(k)
					
					tID := lr.TraceID().String()
					if tID == "" { 
						// Catch anonymous system errors (KAFKA, HIKARI, OOM...)
						if lr.SeverityText() == "ERROR" || lr.SeverityText() == "FATAL" || lr.SeverityText() == "WARN" {
							timeStr := lr.Timestamp().AsTime().Local().Format("15:04:05") 
							sysMsg := fmt.Sprintf("[%s] [%s] %s", timeStr, shortScope, lr.Body().AsString())
							
							mu.Lock()
							if len(systemLogsBuffer) >= MAX_RECENT_TRACES {
								systemLogsBuffer = systemLogsBuffer[1:]
							}
							systemLogsBuffer = append(systemLogsBuffer, sysMsg)
							mu.Unlock()
						}
						continue 
					}

					if lr.SeverityText() == "ERROR" || lr.SeverityText() == "FATAL" {
						mu.Lock()
						tr := getOrCreateTrace(tID)
						if tr.SnapshotRAM == 0 {
							latestMetrics.mu.Lock()
							tr.SnapshotCPU = latestMetrics.CPUUsage
							tr.SnapshotRAM = latestMetrics.RAMUsed
							latestMetrics.mu.Unlock()
						}


						errMsg := fmt.Sprintf("**Message:** %s\n", lr.Body().Str())
						if stackVal, ok := lr.Attributes().Get("exception.stacktrace"); ok {
							errMsg += fmt.Sprintf("**Stacktrace:**\n```text\n%s\n```\n", filterStacktrace(stackVal.Str()))
						}
						tr.ErrorMsgs = append(tr.ErrorMsgs, errMsg)
						
						if !isDebouncing {
							isDebouncing = true
							// Trigger when backend error occurs
							triggerE2EPromptCreation(tID)
						}
						mu.Unlock()
					} else if lr.SeverityText() == "INFO" || lr.SeverityText() == "DEBUG" || lr.SeverityText() == "WARN" {
						tr := getOrCreateTrace(tID)
						
						// Insert shortScope variable to use it
						logMsg := fmt.Sprintf("[%s] [%s] %s", lr.SeverityText(), shortScope, lr.Body().AsString())
						
						tr.Logs = append(tr.Logs, logMsg)
					}
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	// API to receive metrics (RAM / CPU) from OTEL agent
	http.HandleFunc("/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := pmetricotlp.NewExportRequest()
		if err := req.UnmarshalProto(body); err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		latestMetrics.mu.Lock()
		defer latestMetrics.mu.Unlock()

		for i := 0; i < req.Metrics().ResourceMetrics().Len(); i++ {
			rm := req.Metrics().ResourceMetrics().At(i)
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				for k := 0; k < sm.Metrics().Len(); k++ {
					m := sm.Metrics().At(k)
					name := m.Name()

					// Internal function to automatically get numbers (whether Int or Double)
					getVal := func(dp pmetric.NumberDataPoint) float64 {
						if dp.ValueType() == pmetric.NumberDataPointValueTypeDouble {
							return dp.DoubleValue()
						}
						return float64(dp.IntValue())
					}

					// Catch CPU
					if name == "process.cpu.usage" || name == "system.cpu.utilization" || name == "jvm.cpu.recent_utilization" {
						if m.Type() == pmetric.MetricTypeGauge && m.Gauge().DataPoints().Len() > 0 {
							latestMetrics.CPUUsage = getVal(m.Gauge().DataPoints().At(0)) * 100
						} else if m.Type() == pmetric.MetricTypeSum && m.Sum().DataPoints().Len() > 0 {
							latestMetrics.CPUUsage = getVal(m.Sum().DataPoints().At(0)) * 100
						}
					}
					
					if name == "db.client.connections.usage" {
						if m.Type() == pmetric.MetricTypeSum && m.Sum().DataPoints().Len() > 0 {
							dp := m.Sum().DataPoints().At(0)
							val := getVal(dp)
							
							if stateVal, ok := dp.Attributes().Get("state"); ok {
								stateStr := stateVal.AsString()
								if stateStr == "active" || stateStr == "used" {
									latestMetrics.DbActiveConn = val
								} else if stateStr == "idle" {
									latestMetrics.DbIdleConn = val
								}
							}
						}
					}

					if name == "hikaricp.connections.active" || name == "hikaricp.connections.usage" {
						if m.Type() == pmetric.MetricTypeGauge && m.Gauge().DataPoints().Len() > 0 {
							latestMetrics.DbActiveConn = getVal(m.Gauge().DataPoints().At(0))
						}
					}
					if name == "hikaricp.connections.idle" {
						if m.Type() == pmetric.MetricTypeGauge && m.Gauge().DataPoints().Len() > 0 {
							latestMetrics.DbIdleConn = getVal(m.Gauge().DataPoints().At(0))
						}
					}

					if name == "jvm.gc.pause" {
						if m.Type() == pmetric.MetricTypeHistogram && m.Histogram().DataPoints().Len() > 0 {
							latestMetrics.GCPauseTime = m.Histogram().DataPoints().At(0).Sum()
						}
					}


					// Catch used RAM
					if name == "jvm.memory.used" {
						if m.Type() == pmetric.MetricTypeGauge && m.Gauge().DataPoints().Len() > 0 {
							latestMetrics.RAMUsed = getVal(m.Gauge().DataPoints().At(0)) / (1024 * 1024)
						} else if m.Type() == pmetric.MetricTypeSum && m.Sum().DataPoints().Len() > 0 {
							latestMetrics.RAMUsed = getVal(m.Sum().DataPoints().At(0)) / (1024 * 1024)
						}
					}

					// Catch max RAM
					if name == "jvm.memory.limit" {
						if m.Type() == pmetric.MetricTypeGauge && m.Gauge().DataPoints().Len() > 0 {
							latestMetrics.RAMMax = getVal(m.Gauge().DataPoints().At(0)) / (1024 * 1024)
						} else if m.Type() == pmetric.MetricTypeSum && m.Sum().DataPoints().Len() > 0 {
							latestMetrics.RAMMax = getVal(m.Sum().DataPoints().At(0)) / (1024 * 1024)
						}
					}
				}
			}
		}
		
		fmt.Println(getServerDict().MetricsUpdated)
		w.WriteHeader(http.StatusOK)
	})

	// Add new API dedicated for frontend (Receive JSON directly, fast and never fails)
	http.HandleFunc("/v1/frontend-spans", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var spans []SpanRecord
		if err := json.NewDecoder(r.Body).Decode(&spans); err != nil {
			return
		}

		mu.Lock()
		for _, span := range spans {
			if len(spanBuffer) >= MaxBufferSize {
				spanBuffer = spanBuffer[1:]
			}
			spanBuffer = append(spanBuffer, span)
			fmt.Printf(getServerDict().ReceivedFrontendSpan+"\n", span.Name, span.TraceID)
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	// API 1: Get list of 20 most recent requests
	http.HandleFunc("/v1/recent-traces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		mu.Lock()
		defer mu.Unlock()

		var summaries []TraceSummary
		seen := make(map[string]bool)

		// Memory saving to find duplicate requests by time
		type TrackedReq struct {
			Method string
			Path   string
			Time   time.Time
		}
		var tracked []TrackedReq

		// Scan Black Box from newest to oldest
		for i := len(spanBuffer) - 1; i >= 0; i-- {
			s := spanBuffer[i]

			// Skip if this TraceID has been taken already
			if seen[s.TraceID] { continue }

			// Only collect Spans that are API calls (have Method and URL)
			url := s.Attributes["http.url"]
			if url == "" { url = s.Attributes["url.full"] }
			if url == "" { url = s.Attributes["url.path"] }

			method := s.Attributes["http.request.method"]
			if method == "" { method = s.Attributes["http.method"] }

			isWorkerJob := false
			if url == "" && method == "" {
				if s.ParentSpanID == "" && s.Name != "click" && s.Name != "EXCEPTION" && !strings.HasPrefix(s.Name, "GET ") {
					
					lowerName := strings.ToLower(s.Name)
					dbSys := s.Attributes["db.system"]
					
					isDbPing := dbSys != "" || strings.Contains(lowerName, "redis") || strings.Contains(lowerName, "hello") || strings.Contains(lowerName, "testdb")
					isMsgPolling := strings.Contains(lowerName, "consume") || strings.Contains(lowerName, "receive") || strings.Contains(lowerName, "ack")
					isWsStats := strings.Contains(lowerName, "websocketmessagebroker") || strings.Contains(lowerName, "health") || strings.Contains(lowerName, "transporthandlingsockjsservice")

					if !isDbPing && !isMsgPolling && !isWsStats {
						isWorkerJob = true
						method = " JOB" 
						url = s.Name
					}
				}
			}

			if (url != "" && method != "" && method != "OPTIONS" && s.Name != "click" && s.Name != "EXCEPTION") || isWorkerJob {				// Filter duplicate frontend vs backend (Zero-Config Mode)
				path := url
				if strings.Contains(path, "://") { // Shorten http://localhost:8080/api... to /api...
					parts := strings.SplitN(path, "/", 4)
					if len(parts) == 4 { path = "/" + parts[3] }
				}

				isDup := false
				for _, tr := range tracked {
					if tr.Method == method && tr.Path == path {
						// Calculate time difference (Less than 3 seconds difference -> Duplicate)
						diff := s.Timestamp.Sub(tr.Time)
						if diff < 0 { diff = -diff }
						if diff < 3*time.Second { 
							isDup = true
							break
						}
					}
				}
				if isDup { continue }
				tracked = append(tracked, TrackedReq{Method: method, Path: path, Time: s.Timestamp})
				// ==========================================

				status := s.Attributes["http.response.status_code"]
				if status == "" { status = s.Attributes["http.status_code"] }

				// Delete to report all network errors (Status 0)
				hasErr := false
				if status != "" && len(status) > 0 {
					if status[0] == '4' || status[0] == '5' || status == "0" {
						hasErr = true
					}
				}

				if isWorkerJob {
					if s.HasError {
						status = "ERR"
						hasErr = true
					} else {
						status = "OK "
					}
				}

				summaries = append(summaries, TraceSummary{
					TraceID:    s.TraceID,
					Time:       s.Timestamp.Format("15:04:05"),
					Method:     method,
					Url:        url,
					StatusCode: status,
					HasError:   hasErr,
				})
				seen[s.TraceID] = true

				// Stop when have 20 items
				if len(summaries) >= MAX_RECENT_TRACES { break }
			}
		}
		if summaries == nil {
			summaries = make([]TraceSummary, 0)
		}
		json.NewEncoder(w).Encode(summaries)
	})

	// API 2: Return prompt for any selected trace
	http.HandleFunc("/v1/prompt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		traceID := r.URL.Query().Get("traceId")
		lang := r.URL.Query().Get("lang") // Get lang parameter
		if lang == "" { lang = "en" }

		if traceID == "" {
			w.Write([]byte(" Please provide traceId"))
			return
		}

		// Call your divine Prompt generation function
		prompt := generateE2EPrompt(traceID, lang)
		w.Write([]byte(prompt))
	})

	// INTERNAL API FOR MCP CONNECTION AND DATA RETRIEVAL
	// SYSTEM INTERFACE & API (PORT 4319)
	go func() {
		mux := http.NewServeMux()
		
		// 1. Door for AI (MCP) - Return raw Text
		mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			mu.Lock()
			prompt := latestPrompt
			mu.Unlock()
			if prompt == "" {
				w.Write([]byte("No errors yet. Please test the application."))
			} else {
				w.Write([]byte(prompt))
			}
		})

			// 3. API to distribute "Divine" Script for Frontend (Zero-Config Way)
		mux.HandleFunc("/trace2prompt.js", func(w http.ResponseWriter, r *http.Request) {
			// Allow script loading from all domains (CORS)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

			// 🌟 SECTION 5 (COMPLETE): VANILLA JS INTEGRATED TEXT, TIMESTAMPS, HEADERS, USER-AGENT
			jsContent, _ := staticFiles.ReadFile("static/trace2prompt.js")
			w.Write([]byte(jsContent))
		})

		// 2. Door for DEV (Web Interface Dark Mode)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			mu.Lock()
			prompt := latestPrompt
			mu.Unlock()

			if prompt == "" {
				prompt = "⏳ Waiting for system to catch errors... (Try calling API that throws 500 error)"
			}

			// HTML + CSS + JS interface embedded directly in Go
			htmlContent, _ := staticFiles.ReadFile("static/index.html")
			w.Write([]byte(htmlContent))
		})

		log.Fatal(http.ListenAndServe(":4319", mux))
	}()

	fmt.Println(getServerDict().StartupMsg)
	fmt.Println(getServerDict().StartupHint)
	log.Fatal(http.ListenAndServe(":4318", nil))

}