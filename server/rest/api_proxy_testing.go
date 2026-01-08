package rest

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//go:embed proxy_test.html
var testPageHTML string

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing
	},
}

// handleProxyTestIndex serves the HTML test page
func (s *ApiModule) handleProxyTestIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(testPageHTML))
}

// handleProxyTestEcho echoes back request with all headers
func (s *ApiModule) handleProxyTestEcho(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	response := map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL.Path,
		"headers": headers,
		"query":   query,
		"body":    string(body),
	}

	jsonify(w, response)
}

// handleProxyTestWebSocket handles WebSocket echo
func (s *ApiModule) handleProxyTestWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Echo back the message
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			break
		}
	}
}

// handleProxyTestSSE sends Server-Sent Events
func (s *ApiModule) handleProxyTestSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send 10 events, one per second
	for i := 1; i <= 10; i++ {
		data := fmt.Sprintf("data: {\"count\": %d, \"timestamp\": \"%s\"}\n\n",
			i, time.Now().Format(time.RFC3339))
		fmt.Fprint(w, data)
		flusher.Flush()

		if i < 10 {
			time.Sleep(1 * time.Second)
		}
	}
}

// handleProxyTestChunked sends chunked response
func (s *ApiModule) handleProxyTestChunked(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send opening bracket
	fmt.Fprint(w, "[")
	flusher.Flush()

	// Send 5 chunks with delays
	for i := 1; i <= 5; i++ {
		if i > 1 {
			fmt.Fprint(w, ",")
		}
		chunk := fmt.Sprintf("{\"chunk\":%d,\"timestamp\":\"%s\"}",
			i, time.Now().Format(time.RFC3339))
		fmt.Fprint(w, chunk)
		flusher.Flush()

		if i < 5 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Send closing bracket
	fmt.Fprint(w, "]")
	flusher.Flush()
}

// handleProxyTestHeaders returns all received headers
func (s *ApiModule) handleProxyTestHeaders(w http.ResponseWriter, r *http.Request) {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	response := map[string]interface{}{
		"headers":         headers,
		"xForwardedHost":  r.Header.Get("X-Forwarded-Host"),
		"xForwardedProto": r.Header.Get("X-Forwarded-Proto"),
		"xForwardedEmail": r.Header.Get("X-Forwarded-Email"),
		"customHeaders":   getCustomHeaders(r),
	}

	jsonify(w, response)
}

// handleProxyTestPostBody tests POST body forwarding
func (s *ApiModule) handleProxyTestPostBody(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"receivedBytes": len(body),
		"contentType":   r.Header.Get("Content-Type"),
		"body":          string(body),
	}

	jsonify(w, response)
}

// handleProxyTestLargeDownload streams a large file
func (s *ApiModule) handleProxyTestLargeDownload(w http.ResponseWriter, r *http.Request) {
	// Parse size parameter (default 10MB, max 100MB)
	sizeStr := r.URL.Query().Get("size")
	sizeMB := 10
	if sizeStr != "" {
		parsed, err := strconv.Atoi(sizeStr)
		if err == nil && parsed > 0 && parsed <= 100 {
			sizeMB = parsed
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=test-%dmb.bin", sizeMB))

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Stream 1MB chunks
	chunk := make([]byte, 1024*1024)
	for i := 0; i < len(chunk); i++ {
		chunk[i] = byte(i % 256)
	}

	for i := 0; i < sizeMB; i++ {
		_, err := w.Write(chunk)
		if err != nil {
			return
		}
		flusher.Flush()
	}
}

// handleProxyTestLargeUpload handles large file uploads
func (s *ApiModule) handleProxyTestLargeUpload(w http.ResponseWriter, r *http.Request) {
	// Read body in chunks without buffering entire file
	totalBytes := int64(0)
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := r.Body.Read(buffer)
		totalBytes += int64(n)

		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Failed to read upload", http.StatusBadRequest)
			return
		}
	}

	response := map[string]interface{}{
		"success":       true,
		"receivedBytes": totalBytes,
		"receivedMB":    float64(totalBytes) / (1024 * 1024),
	}

	jsonify(w, response)
}

// handleProxyTestTimeout simulates slow backend
func (s *ApiModule) handleProxyTestTimeout(w http.ResponseWriter, r *http.Request) {
	// Parse delay parameter (default 5s, max 30s)
	delayStr := r.URL.Query().Get("delay")
	delaySec := 5
	if delayStr != "" {
		parsed, err := strconv.Atoi(delayStr)
		if err == nil && parsed > 0 && parsed <= 30 {
			delaySec = parsed
		}
	}

	startTime := time.Now()
	time.Sleep(time.Duration(delaySec) * time.Second)
	elapsed := time.Since(startTime)

	response := map[string]interface{}{
		"requestedDelay": delaySec,
		"actualDelay":    elapsed.Seconds(),
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	jsonify(w, response)
}

// handleProxyTestAuth returns authentication information
func (s *ApiModule) handleProxyTestAuth(w http.ResponseWriter, r *http.Request) {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		// Return anonymous status
		response := map[string]interface{}{
			"authenticated":   false,
			"headers":         headers,
			"xForwardedEmail": r.Header.Get("X-Forwarded-Email"),
		}
		w.WriteHeader(http.StatusOK)
		jsonify(w, response)
		return
	}

	// Return authenticated status
	response := map[string]interface{}{
		"authenticated":   true,
		"userEmail":       user.Email,
		"userId":          user.Id,
		"isAdmin":         user.IsAdmin,
		"headers":         headers,
		"xForwardedEmail": r.Header.Get("X-Forwarded-Email"),
	}

	jsonify(w, response)
}

// getCustomHeaders extracts non-standard headers
func getCustomHeaders(r *http.Request) map[string]string {
	custom := make(map[string]string)
	standardHeaders := map[string]bool{
		"Accept": true, "Accept-Encoding": true, "Accept-Language": true,
		"Cache-Control": true, "Connection": true, "Content-Length": true,
		"Content-Type": true, "Cookie": true, "Host": true, "User-Agent": true,
		"Referer": true, "Origin": true,
	}

	for k, v := range r.Header {
		if !standardHeaders[k] && len(v) > 0 {
			custom[k] = v[0]
		}
	}

	return custom
}

// handleProxyTestStatus returns specific HTTP status codes
func (s *ApiModule) handleProxyTestStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Query().Get("code")
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		code = http.StatusBadRequest
	}

	// Set the requested status code
	w.WriteHeader(code)

	// Write a body so we can verify the proxy passes error bodies too
	w.Write([]byte(fmt.Sprintf("Status: %d", code)))
}

// handleProxyTestCookies sets multiple cookies to test header folding
func (s *ApiModule) handleProxyTestCookies(w http.ResponseWriter, r *http.Request) {
	// Set two distinct cookies.
	// Calling SetCookie multiple times generates multiple "Set-Cookie" headers.
	http.SetCookie(w, &http.Cookie{Name: "test-a", Value: "1", Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "test-b", Value: "2", Path: "/"})

	response := map[string]interface{}{
		"cookiesSent": []string{"test-a=1", "test-b=2"},
		"message":     "Check if your proxy merged these headers or dropped one",
	}
	jsonify(w, response)
}

// handleProxyTestCompression returns gzipped content
func (s *ApiModule) handleProxyTestCompression(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")

	// Create gzip writer wrapping the response writer
	gw := gzip.NewWriter(w)
	defer gw.Close()

	// Create data that compresses well (lots of repetition)
	largeString := ""
	for i := 0; i < 1000; i++ {
		largeString += "proxy_compression_test_"
	}

	data := map[string]interface{}{
		"info":    "This content is compressed with gzip",
		"payload": largeString,
	}

	// Write JSON directly to the gzip writer
	json.NewEncoder(gw).Encode(data)
}

// RegisterTestEndpoints registers proxy testing endpoints (no Host requirement)
func (a *ApiModule) RegisterTestEndpoints(r *mux.Router) {
	// Core tests
	r.Methods("GET").Path("/proxy-test").HandlerFunc(a.handleProxyTestIndex)
	r.Methods("GET").Path("/proxy-test/").HandlerFunc(a.handleProxyTestIndex)
	r.Methods("GET", "POST").PathPrefix("/proxy-test/echo").HandlerFunc(a.handleProxyTestEcho)
	r.Methods("GET").Path("/proxy-test/websocket").HandlerFunc(a.handleProxyTestWebSocket)
	r.Methods("GET").Path("/proxy-test/sse").HandlerFunc(a.handleProxyTestSSE)
	r.Methods("GET").Path("/proxy-test/chunked").HandlerFunc(a.handleProxyTestChunked)
	r.Methods("GET").Path("/proxy-test/headers").HandlerFunc(a.handleProxyTestHeaders)

	// Additional tests
	r.Methods("POST").Path("/proxy-test/post-body").HandlerFunc(a.handleProxyTestPostBody)
	r.Methods("GET").Path("/proxy-test/large-download").HandlerFunc(a.handleProxyTestLargeDownload)
	r.Methods("POST").Path("/proxy-test/large-upload").HandlerFunc(a.handleProxyTestLargeUpload)
	r.Methods("GET").Path("/proxy-test/timeout").HandlerFunc(a.handleProxyTestTimeout)
	r.Methods("GET").Path("/proxy-test/auth").HandlerFunc(a.handleProxyTestAuth)

	r.Methods("GET").Path("/proxy-test/status").HandlerFunc(a.handleProxyTestStatus)
	r.Methods("GET").Path("/proxy-test/cookies").HandlerFunc(a.handleProxyTestCookies)
	r.Methods("GET").Path("/proxy-test/compression").HandlerFunc(a.handleProxyTestCompression)
}
