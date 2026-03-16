@echo off
echo ========================================================
echo 🚀 STARTING SPRING BOOT WITH TRACE2PROMPT (DEV MODE)
echo ========================================================

:: --- 0. TỰ ĐỘNG TẢI OTEL AGENT NẾU CHƯA CÓ ---
set OTEL_JAR=opentelemetry-javaagent.jar
if not exist "%OTEL_JAR%" (
    echo [INFO] OpenTelemetry Agent not found!
    echo [INFO] Downloading the latest version automatically...
    curl -L -o %OTEL_JAR% "https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar"
    echo [SUCCESS] Download complete!
) else (
    echo [SUCCESS] OpenTelemetry Agent is ready.
)

echo.

:: --- 1. CẤU HÌNH ĐỊNH TUYẾN DỮ LIỆU ---
set OTEL_SERVICE_NAME=your-service-name
set OTEL_TRACES_EXPORTER=otlp
set OTEL_LOGS_EXPORTER=otlp
set OTEL_METRICS_EXPORTER=otlp
set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
set OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf

:: --- 2. CẤU HÌNH BẮT HEADERS (Để lấy key của Token/Cookie, không lấy value) ---
set OTEL_INSTRUMENTATION_HTTP_CAPTURE_HEADERS_SERVER_REQUEST=Authorization,Cookie,Accept,User-Agent,Content-Type
set OTEL_INSTRUMENTATION_HTTP_SERVER_CAPTURE_REQUEST_HEADERS=Authorization,Cookie,Accept,User-Agent,Content-Type

:: --- 3. CẤU HÌNH TỐC ĐỘ GỬI DỮ LIỆU (500ms) ---
set OTEL_BSP_SCHEDULE_DELAY=500
set OTEL_BLRP_SCHEDULE_DELAY=500

:: --- 4. GẮN AGENT VÀ KHỞI CHẠY ---
set MAVEN_OPTS="-javaagent:%OTEL_JAR%"

echo ⏳ Compiling and Starting application via Maven...
mvn spring-boot:run

pause