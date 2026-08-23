package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandlerRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
	}{
		{name: "root", method: http.MethodGet, path: "/", statusCode: http.StatusOK},
		{name: "liveness", method: http.MethodGet, path: "/healthz", statusCode: http.StatusOK},
		{name: "readiness", method: http.MethodGet, path: "/readyz", statusCode: http.StatusOK},
		{name: "not ready", method: http.MethodGet, path: "/not-ready", statusCode: http.StatusServiceUnavailable},
		{name: "rest status", method: http.MethodGet, path: "/api/status", statusCode: http.StatusOK},
		{name: "rest items", method: http.MethodGet, path: "/api/items", statusCode: http.StatusOK},
		{name: "unknown", method: http.MethodGet, path: "/unknown", statusCode: http.StatusNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/readyz", statusCode: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			responseRecorder := httptest.NewRecorder()

			newHandlerWithConfig(handlerConfig{sseInterval: time.Millisecond}).ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != test.statusCode {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, test.statusCode)
			}
		})
	}
}

func TestStaticAssets(t *testing.T) {
	for _, test := range []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html; charset=utf-8", contains: "Testkit is running"},
		{path: "/static/style.css", contentType: "text/css; charset=utf-8", contains: "font-family"},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusOK)
			}
			if got := responseRecorder.Header().Get("Content-Type"); got != test.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, test.contentType)
			}
			if !strings.Contains(responseRecorder.Body.String(), test.contains) {
				t.Fatalf("body does not contain %q: %s", test.contains, responseRecorder.Body.String())
			}
		})
	}
}

func TestRESTEcho(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"hello":"world"}`))
	request.Header.Set("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d: %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	var payload struct {
		Data   map[string]string `json:"data"`
		Method string            `json:"method"`
		Path   string            `json:"path"`
	}
	if err := json.NewDecoder(responseRecorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode REST response: %v", err)
	}
	if payload.Data["hello"] != "world" || payload.Method != http.MethodPost || payload.Path != "/api/echo" {
		t.Fatalf("unexpected REST payload: %#v", payload)
	}
}

func TestRESTEchoRejectsInvalidAndOversizedJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
	}{
		{name: "invalid", body: "{", statusCode: http.StatusBadRequest},
		{name: "oversized", body: `{"data":"` + strings.Repeat("a", maxJSONBodyBytes) + `"}`, statusCode: http.StatusRequestEntityTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(test.body))
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != test.statusCode {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, test.statusCode)
			}
		})
	}
}

func TestGraphQLTransportAndVariables(t *testing.T) {
	tests := []struct {
		name      string
		request   string
		wantData  map[string]interface{}
		wantError bool
	}{
		{
			name:     "fields",
			request:  `{"query":"{ status version echo(message: \"hello\") }"}`,
			wantData: map[string]interface{}{"status": "ok", "version": version, "echo": "hello"},
		},
		{
			name:     "variables",
			request:  `{"query":"query Echo($message: String!) { echo(message: $message) }","variables":{"message":"variable"}}`,
			wantData: map[string]interface{}{"echo": "variable"},
		},
		{
			name:      "invalid query",
			request:   `{"query":"{ unknown }"}`,
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(test.request))
			request.Header.Set("Content-Type", "application/json")
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d: %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
			}
			var payload struct {
				Data   map[string]interface{} `json:"data"`
				Errors []interface{}          `json:"errors"`
			}
			if err := json.NewDecoder(responseRecorder.Body).Decode(&payload); err != nil {
				t.Fatalf("decode GraphQL response: %v", err)
			}
			if test.wantError {
				if len(payload.Errors) == 0 {
					t.Fatalf("expected GraphQL errors, got %#v", payload)
				}
				return
			}
			for key, want := range test.wantData {
				if payload.Data[key] != want {
					t.Fatalf("data[%q] = %#v, want %#v", key, payload.Data[key], want)
				}
			}
		})
	}
}

func TestGraphQLMethodAndBodyLimits(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	responseRecorder := httptest.NewRecorder()
	newHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /graphql status code = %d, want %d", responseRecorder.Code, http.StatusMethodNotAllowed)
	}

	request = httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{"query":"`+strings.Repeat("a", maxGraphQLBodyBytes)+`"}`))
	responseRecorder = httptest.NewRecorder()
	newHandler().ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized GraphQL status code = %d, want %d", responseRecorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestSSETransportFlushesAndRemainsOpen(t *testing.T) {
	server := httptest.NewServer(newHandlerWithConfig(handlerConfig{sseInterval: 10 * time.Millisecond}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/events", nil)
	if err != nil {
		t.Fatalf("create SSE request: %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	for _, want := range []string{"id: 1", "event: status"} {
		select {
		case line := <-lines:
			if line != want {
				t.Fatalf("SSE line = %q, want %q", line, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for SSE line %q", want)
		}
	}
	select {
	case line := <-lines:
		if line != "data: {\"status\":\"ok\",\"version\":\"devel\"}" {
			t.Fatalf("SSE data line = %q", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE data")
	}

	select {
	case <-lines:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not deliver an incremental event")
	}
}

func TestWebSocketEchoAndBroadcast(t *testing.T) {
	server := httptest.NewServer(newHandler())
	t.Cleanup(server.Close)
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	first, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("dial first WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })
	second, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("dial second WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if err := first.WriteJSON(map[string]string{"message": "hello"}); err != nil {
		t.Fatalf("write WebSocket message: %v", err)
	}
	for name, connection := range map[string]*websocket.Conn{"first": first, "second": second} {
		t.Run(name, func(t *testing.T) {
			_ = connection.SetReadDeadline(time.Now().Add(time.Second))
			var message wsMessage
			if err := connection.ReadJSON(&message); err != nil {
				t.Fatalf("read WebSocket message: %v", err)
			}
			if message.Message != "hello" || message.Version != version {
				t.Fatalf("unexpected WebSocket message: %#v", message)
			}
		})
	}
}

func TestOutbound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("controlled-upstream"))
	}))
	t.Cleanup(upstream.Close)

	request := httptest.NewRequest(http.MethodGet, "/outbound?url="+url.QueryEscape(upstream.URL), nil)
	responseRecorder := httptest.NewRecorder()
	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d: %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	var payload response
	if err := json.NewDecoder(responseRecorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode outbound response: %v", err)
	}
	if payload.UpstreamStatus != http.StatusOK || payload.UpstreamBody != "controlled-upstream" {
		t.Fatalf("unexpected outbound response: %#v", payload)
	}
}

func TestOutboundRejectsInvalidURL(t *testing.T) {
	for _, rawURL := range []string{"", "relative", "file:///etc/passwd"} {
		t.Run(strings.ReplaceAll(rawURL, "/", "_"), func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/outbound?url="+url.QueryEscape(rawURL), nil)
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusBadRequest {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestDecodeJSONRejectsTrailingValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"first":1}{"second":2}`))
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
	}
}

func TestSSEIntervalFromEnv(t *testing.T) {
	t.Setenv("SSE_INTERVAL", "25ms")
	if got := sseIntervalFromEnv(); got != 25*time.Millisecond {
		t.Fatalf("SSE interval = %s, want 25ms", got)
	}
	t.Setenv("SSE_INTERVAL", "invalid")
	if got := sseIntervalFromEnv(); got != defaultSSEInterval {
		t.Fatalf("invalid SSE interval = %s, want %s", got, defaultSSEInterval)
	}
}

func TestGraphQLRejectsInvalidJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader("{"))
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
	}
}

func TestStaticPathTraversalIsNotServed(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/static/../go.mod", nil)
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusTemporaryRedirect)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/go.mod" {
		t.Fatalf("Location = %q, want /go.mod", got)
	}
	body, err := io.ReadAll(responseRecorder.Result().Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if strings.Contains(string(body), "module ") {
		t.Fatal("path traversal exposed repository content")
	}
}
