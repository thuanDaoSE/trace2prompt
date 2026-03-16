package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
)

func runMCPServer() {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer to prevent Antigravity from sending extremely large messages
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(line), &req); err != nil { continue }

		method, _ := req["method"].(string)
		id := req["id"]
		if method == "" || id == nil { continue }

		var result interface{}
		var errObj interface{}

		switch method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]string{"name": "trace2prompt-mcp", "version": "1.0"},
			}
		case "tools/list":
			result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name": "get_latest_trace",
						"description": "GET LATEST ERROR TRACE.",
						"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
					},
				},
			}
		case "tools/call":
			// 🌟 ADD 3 SECOND TIMEOUT TO PREVENT HANGING 
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get("http://localhost:4319/latest")
			var text string
			if err != nil {
				text = "⚠️ Trace2Prompt Server not responding. Please check if you have enabled Trace2Prompt.exe running in background, or press Enter on the black screen if it's frozen."
			} else {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				text = string(body)
			}
			result = map[string]interface{}{
				"content": []map[string]interface{}{{"type": "text", "text": text}},
			}
		default:
			// 🌟 PREVENT RESPONDING TO STRANGE COMMANDS TO AVOID ANTIGRAVITY TIMEOUT
			errObj = map[string]interface{}{"code": -32601, "message": "Method not found"}
		}

		// WRAP UP AND RETURN TO AI
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
		}
		if result != nil {
			resp["result"] = result
		} else if errObj != nil {
			resp["error"] = errObj
		} else {
			resp["result"] = map[string]interface{}{}
		}

		out, _ := json.Marshal(resp)
		fmt.Println(string(out))
	}
}