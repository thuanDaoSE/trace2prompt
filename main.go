package main

import (
	"embed"
	"bytes"
	"flag"
	"fmt"
	"os/exec"
	"runtime"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

const ProjectNamespace = "com.coffeeshop"
const MaxBufferSize = 2000

type SpanRecord struct {
	TraceID    string
	SpanID     string
	Name       string

	ParentSpanID string
	Timestamp  time.Time
	Attributes map[string]string
	HasError   bool
}

type TraceSummary struct {
	TraceID    string `json:"traceId"`
	Time       string `json:"time"`
	Method     string `json:"method"`
	Url        string `json:"url"`
	StatusCode string `json:"statusCode"`
	HasError   bool   `json:"hasError"`
}

var latestMetrics = struct {
	CPUUsage float64
	RAMUsed  float64
	RAMMax   float64
	DbActiveConn float64
	DbIdleConn   float64
	GCPauseTime  float64
	mu       sync.Mutex
}{}

type SpanNode struct {
	SpanID        string
	ParentSpanID  string
	Name          string
	DurationMs    int64
	SQL           string
	MsgSystem     string
	DbSystem      string
	ServiceName   string

	MsgOperation  string 
	MsgDestName   string 

	ExtHttpUrl    string
	ExtHttpMethod string
	Children      []*SpanNode

	RpcSystem        string
	FaasTrigger      string
	GraphqlOperation string
}

type TraceRecord struct {
	TraceID     string
	Spans       map[string]*SpanNode
	ErrorMsgs   []string
	Logs        []string
	Breadcrumbs []string
	ServiceName string
	HttpMethod  string
	HttpUrl     string
	HttpStatus  string
	UserAgent   string
	AuthToken   string
	OsDesc      string
	JavaVersion string
	DbSystem    string
	DbAddress   string

	SnapshotCPU float64
	SnapshotRAM float64

	AuthContext string

	K8sPodName  string
	ContainerID string
	Printed     bool
	LastUpdated time.Time
}

var (
	mu               sync.Mutex
	traceMap         = make(map[string]*TraceRecord)
	isDebouncing     = false
	latestPrompt     = ""
	spanBuffer       = make([]SpanRecord, 0, MaxBufferSize)
	systemLogsBuffer = make([]string, 0, 20)
)

func formatSQL(rawSQL string) string {
	sql := rawSQL
	lowerSQL := strings.ToLower(sql)
	if strings.HasPrefix(lowerSQL, "select ") || strings.HasPrefix(lowerSQL, "insert ") || strings.HasPrefix(lowerSQL, "update ") || strings.HasPrefix(lowerSQL, "delete ") {
		sql = strings.ReplaceAll(sql, " from ", "\n        FROM ")
		sql = strings.ReplaceAll(sql, " where ", "\n        WHERE ")
		sql = strings.ReplaceAll(sql, " left join ", "\n        LEFT JOIN ")
		sql = strings.ReplaceAll(sql, " inner join ", "\n        INNER JOIN ")
		sql = strings.ReplaceAll(sql, " order by ", "\n        ORDER BY ")
		sql = strings.ReplaceAll(sql, " group by ", "\n        GROUP BY ")
	}
	return sql
}

func addSpanToBuffer(span SpanRecord) {
	mu.Lock()
	defer mu.Unlock()
	if len(spanBuffer) >= MaxBufferSize {
		spanBuffer = spanBuffer[1:]
	}
	spanBuffer = append(spanBuffer, span)
}

func copyToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}
	in, err := cmd.StdinPipe()
	if err != nil { return }
	go func() {
		defer in.Close()
		in.Write([]byte(text))
	}()
	cmd.Run()
}

func getOrCreateTrace(traceID string) *TraceRecord {
	if tr, exists := traceMap[traceID]; exists {
		tr.LastUpdated = time.Now()
		return tr
	}
	tr := &TraceRecord{
		TraceID:     traceID,
		Spans:       make(map[string]*SpanNode),
		LastUpdated: time.Now(),
	}
	traceMap[traceID] = tr
	return tr
}

func buildTreeStr(node *SpanNode, depth int, sb *strings.Builder) {
	indent := strings.Repeat("  ", depth)
	prefix := indent

	serviceTag := ""
	if node.ServiceName != "" {
		serviceTag = fmt.Sprintf("📦 [%s] ", node.ServiceName)
	}

	if node.ExtHttpUrl != "" {
		urlLower := strings.ToLower(node.ExtHttpUrl)
		if strings.Contains(urlLower, "keycloak") || strings.Contains(urlLower, "auth0") || strings.Contains(urlLower, "cognito") {
			sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s `%s %s`\n", prefix, node.DurationMs, serviceTag, getServerDict().IamAuth, node.ExtHttpMethod, node.ExtHttpUrl))
			return // Stop here, don't print the default EXTERNAL API block below
		}
	}

	icon := "⚙️"
	systemName := ""

	// --- BROKERS ---
	if node.MsgSystem == "kafka" { icon = "📨"; systemName = "[KAFKA]" }
	if node.MsgSystem == "rabbitmq" { icon = "🐰"; systemName = "[RABBITMQ]" }
	if node.MsgSystem == "aws_sqs" || node.MsgSystem == "sqs" { icon = "☁️"; systemName = "[AWS SQS]" }

	// --- DATABASES & CACHE ---
	if node.DbSystem == "redis" || node.DbSystem == "memcached" { icon = "⚡"; systemName = "[CACHE]" }
	nodeNameUpper := strings.ToUpper(node.Name)
	if strings.Contains(nodeNameUpper, "S3.") || strings.Contains(nodeNameUpper, "S3 ") {
		icon = "🪣"; systemName = "[AWS S3]"
	}
	if strings.Contains(nodeNameUpper, "DYNAMODB") { icon = "📚"; systemName = "[DYNAMODB]" }
	if strings.Contains(nodeNameUpper, "S3.") || node.RpcSystem == "aws-api" {
		icon = "☁️"; systemName = "[AWS/S3]"
	}
	
	if node.DbSystem == "postgresql" || node.DbSystem == "mysql" || node.DbSystem == "mssql" || node.DbSystem == "oracle" { icon = "🗄️"; systemName = "[SQL DB]" }
	if node.DbSystem == "mongodb" { icon = "🍃"; systemName = "[MONGODB]" }
	if node.DbSystem == "elasticsearch" { icon = "🔍"; systemName = "[ELASTICSEARCH]" }

	if node.RpcSystem == "grpc" { icon = "🚀"; systemName = "[gRPC]" }
	if node.RpcSystem == "aws-api" || strings.Contains(strings.ToUpper(node.Name), "S3.") { icon = "☁️"; systemName = "[AWS S3]" }
	if node.FaasTrigger != "" { icon = "🌩️"; systemName = "[SERVERLESS]" }
	if node.GraphqlOperation != "" { icon = "⚛️"; systemName = "[GRAPHQL]" }

	if node.SQL != "" {
		if node.GraphqlOperation != "" {
			sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s %s %s\n%s    ```graphql\n%s    %s\n%s    ```\n", prefix, node.DurationMs, serviceTag, icon, systemName, getServerDict().GraphqlOp, prefix, prefix, node.SQL, prefix))
		} else if node.DbSystem == "redis" || node.DbSystem == "memcached" {
		
			sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s %s %s\n%s    ```text\n%s    %s\n%s    ```\n", prefix, node.DurationMs, serviceTag, icon, systemName, getServerDict().RedisCmd, prefix, prefix, node.SQL, prefix))
		} else if node.DbSystem == "mongodb" || node.DbSystem == "elasticsearch" {
			// NoSQL use JSON + Pretty Print
			prettyJson := node.SQL
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(node.SQL), "", "  "); err == nil {
				prettyJson = buf.String()
			}
			prettyJson = strings.ReplaceAll(prettyJson, "\n", "\n"+prefix+"    ")
			sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s %s %s\n%s    ```json\n%s    %s\n%s    ```\n", prefix, node.DurationMs, serviceTag, icon, systemName, getServerDict().MongoQuery, prefix, prefix, prettyJson, prefix))
			
		} else {
			// SQL Relational
			if icon == "⚙️" { icon = "🗄️"; systemName = "[DB]" } // Fallback
			sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s %s %s\n%s    ```sql\n%s    %s\n%s    ```\n", prefix, node.DurationMs, serviceTag, icon, systemName, getServerDict().SqlQuery, prefix, prefix, formatSQL(node.SQL), prefix))
		}
	} else if node.ExtHttpUrl != "" {
		sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s `%s %s`\n", prefix, node.DurationMs, serviceTag, getServerDict().ExternalApi, node.ExtHttpMethod, node.ExtHttpUrl))
	} else {
		displayName := node.Name
		if systemName != "" { displayName = systemName + " " + node.Name }
		sb.WriteString(fmt.Sprintf("%s- [%d ms] %s%s `%s`\n", prefix, node.DurationMs, serviceTag, icon, displayName))
	}
	
	// Display additional Broker information
	if node.MsgSystem != "" {
		action := getServerDict().InteractWith
		if node.MsgOperation != "" { action = node.MsgOperation }
		topic := getServerDict().UnknownTopic
		if node.MsgDestName != "" { topic = node.MsgDestName }
		
		sb.WriteString(fmt.Sprintf("%s  - %s %s `%s`\n", indent, action, getServerDict().TopicQueue, topic))
	}

	var repeatCount int = 0
	for i := 0; i < len(node.Children); i++ {
		child := node.Children[i]
		
		if i < len(node.Children)-1 {
			nextChild := node.Children[i+1]
			if child.SQL != "" && child.SQL == nextChild.SQL && child.Name == nextChild.Name {
				repeatCount++
				continue
			}
		}

		buildTreeStr(child, depth+1, sb)
		
		if repeatCount > 0 {
			indentChild := strings.Repeat("  ", depth+1)
			sb.WriteString(fmt.Sprintf(getServerDict().N1DetectedBuild, indentChild, repeatCount+1))
			repeatCount = 0
		}
	}
}

func triggerE2EPromptCreation(traceID string) {
	tr := getOrCreateTrace(traceID)
	// 🌟 SKIP OPTIONS REQUESTS (CORS Preflight)
	if tr.HttpMethod == "OPTIONS" {
		return 
	}

	go func() {
		time.Sleep(2000 * time.Millisecond) 
		
		prompt := generateE2EPrompt(traceID, "en")
		if prompt != "" {
			mu.Lock()
			latestPrompt = prompt
			mu.Unlock()
			fmt.Printf(getServerDict().NewErrorCaught+"\n", traceID)
		}
		
		mu.Lock()
		isDebouncing = false
		mu.Unlock()
	}()
}

func main() {
	var mcpMode bool
	var langFlag string
	flag.BoolVar(&mcpMode, "mcp", false, "Run in MCP Server mode for AI Agent")
	flag.StringVar(&langFlag, "lang", "en", "Language for server messages and prompts: en or vi")
	flag.Parse()

	// Apply language setting globally
	serverLang = langFlag

	if mcpMode {
		runMCPServer()
		return 
	}

	startServers()
}