
console.log(T2P_STRINGS[(window.TRACE2PROMPT_LANG || 'en').toLowerCase()]?.initStart || T2P_STRINGS['en'].initStart);

// =============================================
// i18n: set window.TRACE2PROMPT_LANG = 'vi'
// before loading this script to use Vietnamese.
// =============================================
const T2P_LANG = (window.TRACE2PROMPT_LANG || 'en').toLowerCase();
const T2P_STRINGS = {
  vi: {
    truncated:      '\n\n... [CẮT BỚT: NỘI DUNG QUÁ DÀI]',
    networkError:   'Lỗi Mạng: ERR_CONNECTION_REFUSED hoặc TIMEOUT. Không thể kết nối tới Server.',
    binaryData:     '[Dữ liệu Nhị phân/Blob]',
    initDone:       '🟢 [Trace2Prompt] E2E Sensor ĐÃ KHỞI TẠO HOÀN TOÀN!',
    initStart:      '🟢 [Trace2Prompt] Đang khởi tạo E2E Sensor (Ultimate Edition)...',
    fileUpload:     '[TẢI FILE LÊN] Tên: ',
    fileType:       ' | Loại: ',
    fileSize:       ' | Kích thước: ',
    object:         '[Đối tượng]',
    wsConnError:    'Lỗi Kết nối WebSocket',
  },
  en: {
    truncated:      '\n\n... [TRUNCATED: CONTENT TOO LARGE]',
    networkError:   'Network Error: ERR_CONNECTION_REFUSED or TIMEOUT. Unable to connect to Server.',
    binaryData:     '[Binary/Blob Data]',
    initDone:       '🟢 [Trace2Prompt] E2E Sensor FULLY INITIALIZED!',
    initStart:      '🟢 [Trace2Prompt] Initializing E2E Sensor (Ultimate Edition)...',
    fileUpload:     '[FILE UPLOAD] Name: ',
    fileType:       ' | Type: ',
    fileSize:       ' | Size: ',
    object:         '[Object]',
    wsConnError:    'WebSocket Connection Error',
  },
};
const T2P = T2P_STRINGS[T2P_LANG] || T2P_STRINGS['en'];

const API_URL = 'http://localhost:4318/v1/frontend-spans';
const originalFetch = window.fetch; 

// 1. List of Golden Keywords (Combined into Regex for ultra-fast scanning)
const SENSITIVE_KEYS = [
    'password', 'passwd', 'pwd', 'secret', 'client_secret', 'app_secret', 
    'token', 'access_token', 'refresh_token', 'auth_token', 'bearer', 
    'api_key', 'apikey', 'private_key', 'public_key', 'credentials', 'session_id', 'cookie',
    'credit_card', 'card_number', 'pan', 'cvv', 'cvc', 'cvn', 'iban', 'account_number', 'routing_number', 'pin',
    'ssn', 'passport', 'cccd', 'id_card', 'phone', 'phone_number', 'mobile', 'email', 'email_address', 'birthdate', 'dob'
];

// 1. Summarize JS Runtime Errors (App crash, Syntax error...)
window.addEventListener('error', function(event) {
    sendSpan('🔥 JS_CRASH', generateId(16), generateId(8), {
        'error.type': 'UnhandledException',
        'error.message': event.message,
        'error.file': event.filename,
        'error.line': String(event.lineno),
        'error.stack': event.error && event.error.stack ? event.error.stack.substring(0, 1000) : '',
        'page.url': window.location.href
    }, true); // Flag true to report red error
});

// 2. Summarize rejected Promise errors (API fetch errors without try/catch)
window.addEventListener('unhandledrejection', function(event) {
    let reason = event.reason;
    let msg = typeof reason === 'object' ? (reason.message || JSON.stringify(reason)) : String(reason);
    sendSpan('⚠️ PROMISE_REJECT', generateId(16), generateId(8), {
        'error.type': 'UnhandledRejection',
        'error.message': msg,
        'page.url': window.location.href
    }, true);
});

// ==========================================
// 1. UPGRADE CONSOLE CAPTURE (LOG, WARN, ERROR) ANTI-SPAM
// ==========================================
const originalConsole = {
    log: console.log,
    warn: console.warn,
    error: console.error
};

['log', 'warn', 'error'].forEach(level => {
    console[level] = function(...args) {
        // Still print to F12 normally
        originalConsole[level].apply(console, args);
        
        let msg = args.map(a => {
            if (a instanceof Error) return a.stack || a.message;
            if (typeof a === 'object') {
                try { return JSON.stringify(a).substring(0, 300); } catch(e) { return T2P.object; }
            }
            return String(a);
        }).join(' ');

        // Filter garbage: Skip tool's own logs and garbage Dev config logs
        if (!msg.includes('trace2prompt') && !msg.includes('sockjs') && msg.trim() !== '') {
            // Limit length to prevent Prompt bloat
            let finalMsg = msg.length > 300 ? msg.substring(0, 300) + '...[TRUNCATED]' : msg;
            
            // Icon for liveliness
            let icon = level === 'error' ? '🚨' : (level === 'warn' ? '⚠️' : 'ℹ️');
            
            sendSpan(`${icon} CONSOLE_${level.toUpperCase()}`, generateId(16), generateId(8), {
                'console.message': finalMsg,
                'page.url': window.location.href
            }, level === 'error'); // Only report red error (activate AI) if it's console.error
        }
    };
});

// ==========================================
// 2. ADD RESOURCE ERROR CAPTURE (Image/Script 404 errors)
// ==========================================
window.addEventListener('error', function(event) {
    // If event contains src or href -> This is an HTML element failed to load resource
    if (event.target && (event.target.src || event.target.href)) {
        const resourceUrl = event.target.src || event.target.href;
        const tagName = event.target.tagName;
        
        sendSpan('🖼️ RESOURCE_ERROR', generateId(16), generateId(8), {
            'error.type': 'ResourceLoadError',
            'error.message': `Failed to load <${tagName.toLowerCase()}> from: ${resourceUrl}`,
            'page.url': window.location.href
        }, true);
    }
}, true); // Must have 'true' flag (Capture phase) to catch HTML element errors


// Create super-strong combined Regex: Find all Keys in the above list
const MASKING_REGEX = new RegExp(`"(${SENSITIVE_KEYS.join('|')})"\\s*:\\s*"[^"]+"`, 'gi');

// Function to calculate Byte size of a string
function getByteSize(str) {
    if (!str) return 0;
    return new Blob([str]).size;
}

// Function to format Bytes to KB, MB for human (and AI) readability
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 🌟 SMART FORM-DATA AND FILE UPLOAD PROCESSING FUNCTION
function extractFormData(body) {
    if (!(body instanceof FormData)) return { safeBody: body, sizeStr: '' };
    
    let obj = {};
    let totalBytes = 0;
    
    for (let [key, value] of body.entries()) {
        if (value instanceof File) {
            // If it's a File -> Only get Metadata, skip content
            obj[key] = `${T2P.fileUpload}${value.name}${T2P.fileType}${value.type}${T2P.fileSize}${formatBytes(value.size)}`;
            totalBytes += value.size;
        } else {
            // If it's normal Text in Form
            obj[key] = value;
            totalBytes += getByteSize(String(value));
        }
    }
    return {
        safeBody: JSON.stringify(obj, null, 2),
        sizeStr: formatBytes(totalBytes)
    };
}


// Body cleaning function
function maskSensitiveBody(bodyStr) {
    if (!bodyStr || typeof bodyStr !== 'string') return bodyStr;
    // Find and mask values, keep Keys to know the field exists
    return bodyStr.replace(MASKING_REGEX, '"$1":"[REDACTED]"');
}

// Auto-format JSON and safe truncation function
function formatAndTruncate(dataStr, maxLen) {
    if (!dataStr) return '';
    let formatted = dataStr;
    try {
        // Try to parse and format JSON (line breaks, 2-space indent)
        let parsed = JSON.parse(dataStr);
        formatted = JSON.stringify(parsed, null, 2);
    } catch(e) {
        // If not JSON (HTML/Text) then keep original string
    }
    
    // Increase limit because formatted JSON generates many spaces and line breaks
    if (formatted.length > maxLen) {
        return formatted.substring(0, maxLen) + T2P.truncated;
    }
    return formatted;
}


function generateId(bytes) {
    const arr = new Uint8Array(bytes);
    crypto.getRandomValues(arr);
    return Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('');
}

function sendSpan(name, traceId, spanId, attributes, hasError = false) {
    originalFetch(API_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify([{
            TraceID: traceId,
            SpanID: spanId,
            Name: name,
            Timestamp: new Date().toISOString(),
            Attributes: attributes,
            HasError: hasError
        }])
    }).catch(e => {});
}

// 🌟 AXIOS INTERCEPTION AT THE ROOT (Including Headers)
const originalOpen = XMLHttpRequest.prototype.open;
const originalSend = XMLHttpRequest.prototype.send;
const originalSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader;

XMLHttpRequest.prototype.open = function(method, url) {
    let urlStr = typeof url === 'string' ? url : (url ? url.toString() : '');
    if (urlStr.includes(':4318')) {
        this._ignoreTrace = true;
        return originalOpen.apply(this, arguments);
    }
    
    this._traceId = generateId(16);
    this._spanId = generateId(8);
    this._method = method;
    this._url = urlStr;
    this._reqHeaders = {}; 

    // NOTE: Must call originalOpen before being allowed to setHeader
    const result = originalOpen.apply(this, arguments);
    
    
    return result;
};

// Intercept Header function to capture Tokens
XMLHttpRequest.prototype.setRequestHeader = function(header, value) {
    if (!this._ignoreTrace) {
        let lowerHeader = header.toLowerCase();
        // Mask if it's Authorization Header or contains token/key/cookie keywords
        if (lowerHeader === 'authorization' || lowerHeader.includes('api-key') || lowerHeader.includes('token') || lowerHeader === 'cookie') {
             this._reqHeaders[header] = '[REDACTED]';
        } else {
             this._reqHeaders[header] = value; 
        }
    }
    return originalSetRequestHeader.apply(this, arguments);
};

XMLHttpRequest.prototype.send = function(body) {
    if (this._ignoreTrace) return originalSend.apply(this, arguments);
    
    let safeBody = "";
    let reqSizeStr = "0 B";
    
    if (body instanceof FormData) {
        let formDataInfo = extractFormData(body);
        safeBody = formDataInfo.safeBody;
        reqSizeStr = formDataInfo.sizeStr;
    } else if (body && typeof body === 'string') {
        reqSizeStr = formatBytes(getByteSize(body));
        safeBody = maskSensitiveBody(body);
    }


    // 🌟 FIX: Also listen to error and timeout events due to network drops
    const handleEnd = () => {
        if (this._hasReported) return;
        this._hasReported = true;

        // Status 0 means network connection error (Timeout, CORS, Server down)
        const isNetworkError = this.status === 0;
        const isHttpError = this.status >= 400;
        const hasError = isNetworkError || isHttpError;

        // 🌟 FIX: Safely read ResponseText (Prevent Crash with Axios)
        let resText = "";
        try {
            if (!this.responseType || this.responseType === 'text') {
                resText = this.responseText;
            } else if (this.responseType === 'json') {
                resText = JSON.stringify(this.response);
            } else {
                resText = T2P.binaryData;
            }
        } catch(e) {}

        let attrs = {
            'http.method': this._method,
            'http.url': this._url,
            'http.status_code': String(this.status),
            'http.request.headers': JSON.stringify(this._reqHeaders),
            'http.request.body': formatAndTruncate(safeBody, 1500),
            'http.response.body': formatAndTruncate(maskSensitiveBody(resText), 1500), 
            'http.user_agent': navigator.userAgent,
            'page.current_url': window.location.href,
            'network.status': navigator.onLine ? 'Online' : 'Offline',
            'device.viewport': `${window.innerWidth}x${window.innerHeight}`,
            'http.request.size': reqSizeStr,
            'http.response.size': formatBytes(getByteSize(resText)), // 🌟 Use safe variable
        };

        if (isNetworkError) {
            attrs['exception.message'] = T2P.networkError;
        }

        sendSpan('HTTP ' + this._method, this._traceId, this._spanId, attrs, hasError);
    };

    this.addEventListener('loadend', handleEnd);
    this.addEventListener('error', handleEnd);
    this.addEventListener('timeout', handleEnd);

    originalSend.apply(this, arguments);
};

// 🌟 FETCH API INTERCEPTION (Including Headers)
window.fetch = async function(...args) {
    let url = args[0];
    let urlStr = typeof url === 'string' ? url : (url ? url.url : '');
    if (urlStr.includes(':4318')) return originalFetch.apply(this, args);

    const traceId = generateId(16), spanId = generateId(8);
    let options = args[1] || {};
    let method = options.method || 'GET';
    
    

    // Safe Header extraction (Your old code remains below)
    let reqHeaders = {};
    if (options.headers) {
        try {
            let h = new Headers(options.headers);
            h.forEach((val, key) => { reqHeaders[key] = val.length > 30 ? val.substring(0, 15) + '...***' : val; });
        } catch(e) {}
    }

    let safeBody = "";
    let reqSizeStr = "0 B";

    if (options.body instanceof FormData) {
        let formDataInfo = extractFormData(options.body);
        safeBody = formDataInfo.safeBody;
        reqSizeStr = formDataInfo.sizeStr;
    } else if (options.body && typeof options.body === 'string') {
        reqSizeStr = formatBytes(getByteSize(options.body));
        safeBody = maskSensitiveBody(options.body);
    }
    
    try {
        const response = await originalFetch.apply(this, args);
        const clonedRes = response.clone();
        clonedRes.text().then(text => {
            sendSpan('HTTP ' + method, traceId, spanId, {
                'http.method': method, 'http.url': urlStr, 'http.status_code': String(response.status),
                'http.request.headers': JSON.stringify(reqHeaders),
                'http.request.body': formatAndTruncate(safeBody, 1500),
                'http.response.body': formatAndTruncate(maskSensitiveBody(text), 1500),
                'http.user_agent': navigator.userAgent,
                // 🌟 ADD 3 GOLDEN ATTRIBUTES HERE:
                'page.current_url': window.location.href, 
                'network.status': navigator.onLine ? 'Online' : 'Offline', 
                'device.viewport': `${window.innerWidth}x${window.innerHeight}`,
                'http.request.size': reqSizeStr,
                'http.response.size': formatBytes(getByteSize(text))
            }, response.status >= 400);
        }).catch(() => {});
        return response;
    } catch (err) {
        sendSpan('HTTP ' + method, traceId, spanId, {
            'http.method': method, 'http.url': urlStr, 'http.status_code': '500',
            'exception.message': err.message, 'http.user_agent': navigator.userAgent
        }, true);
        throw err;
    }
};

    // ==========================================
    // 🌟 NEW WEAPON: WEBSOCKET INTERCEPTION
    // ==========================================
    const OriginalWebSocket = window.WebSocket;
    window.WebSocket = function(url, protocols) {
        const ws = new OriginalWebSocket(url, protocols);
        const wsTraceId = generateId(16); // Create 1 TraceID for the entire connection session

        // Internal function to send WebSocket Span to Backend
        function reportWsEvent(op, payload, isError) {
            const spanId = generateId(8);
            let safePayload = maskSensitiveBody(typeof payload === 'string' ? payload : T2P.binaryData);
            
            let span = {
                TraceID: wsTraceId,
                SpanID: spanId,
                Name: `WS ${op}`,
                Timestamp: new Date().toISOString(),
                HasError: isError,
                Attributes: {
                    'http.url': typeof url === 'string' ? url : url.url,
                    'messaging.system': 'websocket',
                    'messaging.operation': op,
                    'messaging.payload': formatAndTruncate(safePayload, 1500),
                    'page.current_url': window.location.href
                }
            };
            // Send quietly to port 4318
            fetch('http://localhost:4318/v1/frontend-spans', {
                method: 'POST',
                body: JSON.stringify([span])
            }).catch(e => {});
        }

        // 1. Catch outgoing messages (Client -> Server)
        const originalSend = ws.send;
        ws.send = function(data) {
            reportWsEvent('SEND', data, false);
            return originalSend.apply(this, arguments);
        };

        // 2. Catch incoming messages (Server -> Client)
        ws.addEventListener('message', function(event) {
            reportWsEvent('RECEIVE', event.data, false);
        });

        // 3. Catch Errors and Connection Close
        ws.addEventListener('error', function(event) {
            reportWsEvent('ERROR', T2P.wsConnError, true);
        });
        ws.addEventListener('close', function(event) {
            reportWsEvent('CLOSE', `Code: ${event.code}`, !event.wasClean);
        });

        return ws;
    };

document.addEventListener('click', (e) => {
    const target = e.target;
    let xpath = target.tagName;
    if (target.id) xpath += '#' + target.id;
    if (target.className && typeof target.className === 'string') xpath += '.' + target.className.replace(/ /g, '.');
    
    // Your excellent code snippet is kept
    let innerText = target.innerText ? target.innerText.trim().substring(0, 30).replace(/\n/g, ' ') : '';
    let elementInfo = (innerText ? '[' + innerText + '] ' : '') + xpath;
    
    sendSpan('click', generateId(16), generateId(8), {
        'target_xpath': elementInfo,
        'http.url': window.location.href
    });
}, true);

console.log(T2P.initDone);