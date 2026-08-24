package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
		{name: "root redirects", method: http.MethodGet, path: "/", statusCode: http.StatusFound},
		{name: "websocket page redirects", method: http.MethodGet, path: "/websocket", statusCode: http.StatusFound},
		{name: "rest lab redirects", method: http.MethodGet, path: "/rest", statusCode: http.StatusFound},
		{name: "graphql lab redirects", method: http.MethodGet, path: "/graphql-lab", statusCode: http.StatusFound},
		{name: "sse lab redirects", method: http.MethodGet, path: "/sse", statusCode: http.StatusFound},
		{name: "english home", method: http.MethodGet, path: "/en/", statusCode: http.StatusOK},
		{name: "english websocket page", method: http.MethodGet, path: "/en/websocket", statusCode: http.StatusOK},
		{name: "english rest lab", method: http.MethodGet, path: "/en/rest", statusCode: http.StatusOK},
		{name: "english graphql lab", method: http.MethodGet, path: "/en/graphql-lab", statusCode: http.StatusOK},
		{name: "english sse lab", method: http.MethodGet, path: "/en/sse", statusCode: http.StatusOK},
		{name: "portuguese home", method: http.MethodGet, path: "/pt-BR/", statusCode: http.StatusOK},
		{name: "spanish websocket page", method: http.MethodGet, path: "/es-AR/websocket", statusCode: http.StatusOK},
		{name: "liveness", method: http.MethodGet, path: "/healthz", statusCode: http.StatusOK},
		{name: "readiness", method: http.MethodGet, path: "/readyz", statusCode: http.StatusOK},
		{name: "not ready", method: http.MethodGet, path: "/not-ready", statusCode: http.StatusServiceUnavailable},
		{name: "rest status", method: http.MethodGet, path: "/api/status", statusCode: http.StatusOK},
		{name: "rest items", method: http.MethodGet, path: "/api/items", statusCode: http.StatusOK},
		{name: "unknown", method: http.MethodGet, path: "/unknown", statusCode: http.StatusNotFound},
		{name: "unknown locale", method: http.MethodGet, path: "/fr/", statusCode: http.StatusNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/readyz", statusCode: http.StatusMethodNotAllowed},
		{name: "websocket page method not allowed", method: http.MethodPost, path: "/websocket", statusCode: http.StatusMethodNotAllowed},
		{name: "localized page method not allowed", method: http.MethodPost, path: "/pt-BR/", statusCode: http.StatusMethodNotAllowed},
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
		{path: "/static/style.css", contentType: "text/css; charset=utf-8", contains: "font-family"},
		{path: "/static/app.js", contentType: "text/javascript; charset=utf-8", contains: "mountWebSocketClient"},
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

func TestDashboardRendersWebSocketConsole(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/en/", nil)
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusOK)
	}
	if got := responseRecorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	for _, want := range []string{
		`<html lang="en">`,
		"Transport console",
		"build devel",
		"Open lab",
		"/en/rest",
		"/en/graphql-lab",
		"/en/sse",
		"/en/websocket",
		"4 active",
		`data-locale="en"`,
		"/static/app.js",
	} {
		if !strings.Contains(responseRecorder.Body.String(), want) {
			t.Fatalf("dashboard does not contain %q", want)
		}
	}
	if got := strings.Count(responseRecorder.Body.String(), "data-ws-client"); got != 0 {
		t.Fatalf("home contains %d WebSocket clients, want 0", got)
	}
}

func TestProtocolLabPagesRenderLocalizedContracts(t *testing.T) {
	tests := []struct {
		path     string
		locale   string
		content  string
		endpoint string
		marker   string
		back     string
	}{
		{path: "/en/rest", locale: "en", content: "REST Lab", endpoint: "/api/*", marker: "data-rest-lab", back: "Back to surface map"},
		{path: "/pt-BR/graphql-lab", locale: "pt-BR", content: "Laboratório GraphQL", endpoint: "/graphql", marker: "data-graphql-lab", back: "Voltar ao mapa de superfícies"},
		{path: "/es-AR/sse", locale: "es-AR", content: "Laboratorio SSE", endpoint: "/events", marker: "data-sse-lab", back: "Volver al mapa de superficies"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			responseRecorder := httptest.NewRecorder()
			newHandler().ServeHTTP(responseRecorder, request)

			body := responseRecorder.Body.String()
			if responseRecorder.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusOK)
			}
			for _, want := range []string{
				`<html lang="` + test.locale + `">`,
				test.content,
				test.endpoint,
				test.marker,
				test.back,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("response does not contain %q", want)
				}
			}
		})
	}
}

func TestGraphQLLabFooterHasSingleSeparator(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/en/graphql-lab", nil)
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if got := strings.Count(responseRecorder.Body.String(), `<span class="footer-separator" aria-hidden="true">/</span>`); got != 1 {
		t.Fatalf("GraphQL lab footer separators = %d, want 1", got)
	}
}

func TestDocumentationListsBrowserLabAliases(t *testing.T) {
	for _, path := range []string{"README.md", "docs/pt-BR/README.md", "docs/es-AR/README.md"} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read documentation: %v", err)
			}
			for _, alias := range []string{"GET /rest", "GET /graphql-lab", "GET /sse"} {
				if !strings.Contains(string(content), alias) {
					t.Fatalf("documentation does not list browser alias %q", alias)
				}
			}
		})
	}
}

func TestWebSocketPageRendersBreadcrumbsAndConsole(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/en/websocket", nil)
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusOK)
	}
	for _, want := range []string{
		"WebSocket",
		"lab.",
		"aria-label=\"Breadcrumb\"",
		"aria-current=\"page\">WebSocket",
		"href=\"/en/\">Home",
		"Back to surface map",
		"build devel",
		"Client A",
		"Client B",
		"/static/app.js",
	} {
		if !strings.Contains(responseRecorder.Body.String(), want) {
			t.Fatalf("WebSocket page does not contain %q", want)
		}
	}
	if got := strings.Count(responseRecorder.Body.String(), "data-ws-client"); got != 2 {
		t.Fatalf("WebSocket page contains %d clients, want 2", got)
	}
}

func TestLocaleAliasRedirectsByPreference(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		acceptLanguage string
		cookie         string
		query          string
		location       string
	}{
		{name: "accept language", path: "/", acceptLanguage: "pt-BR, en;q=0.8", location: "/pt-BR/"},
		{name: "cookie wins over header", path: "/websocket", acceptLanguage: "en", cookie: "es-AR", location: "/es-AR/websocket"},
		{name: "lab alias uses preference", path: "/rest", acceptLanguage: "pt-BR", location: "/pt-BR/rest"},
		{name: "query wins over cookie", path: "/", acceptLanguage: "en", cookie: "pt-BR", query: "?lang=es-AR", location: "/es-AR/"},
		{name: "fallback", path: "/", acceptLanguage: "de-DE", location: "/en/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path+test.query, nil)
			request.Header.Set("Accept-Language", test.acceptLanguage)
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "testkit_locale", Value: test.cookie})
			}
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusFound {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusFound)
			}
			if got := responseRecorder.Header().Get("Location"); got != test.location {
				t.Fatalf("Location = %q, want %q", got, test.location)
			}
			if got := responseRecorder.Header().Values("Vary"); len(got) != 2 {
				t.Fatalf("Vary = %q, want Accept-Language and Cookie", got)
			}
		})
	}
}

func TestLocalizedPagesExposeLocaleAndLanguageLinks(t *testing.T) {
	tests := []struct {
		path       string
		locale     string
		content    string
		otherRoute string
		breadcrumb string
	}{
		{path: "/pt-BR/", locale: "pt-BR", content: "Console de transporte", otherRoute: "/pt-BR/websocket"},
		{path: "/es-AR/websocket", locale: "es-AR", content: "Laboratorio WebSocket", otherRoute: "/es-AR/", breadcrumb: "Ruta de navegación"},
	}

	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			responseRecorder := httptest.NewRecorder()

			newHandler().ServeHTTP(responseRecorder, request)

			body := responseRecorder.Body.String()
			if responseRecorder.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", responseRecorder.Code, http.StatusOK)
			}
			if responseRecorder.Header().Get("Content-Language") != test.locale {
				t.Fatalf("Content-Language = %q, want %q", responseRecorder.Header().Get("Content-Language"), test.locale)
			}
			for _, want := range []string{
				`<html lang="` + test.locale + `">`,
				test.content,
				`data-locale="` + test.locale + `"`,
				test.otherRoute,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("response does not contain %q", want)
				}
			}
			if test.breadcrumb != "" && !strings.Contains(body, `aria-label="`+test.breadcrumb+`"`) {
				t.Fatalf("localized breadcrumb label %q not found", test.breadcrumb)
			}
			if cookie := responseRecorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "testkit_locale="+test.locale) {
				t.Fatalf("Set-Cookie = %q, want locale cookie", cookie)
			}
		})
	}
}

func TestTranslationCatalogsHaveSameKeys(t *testing.T) {
	english := pageTranslationCatalog.values[localeEN]
	for _, currentLocale := range supportedLocales {
		translations := pageTranslationCatalog.values[currentLocale]
		if len(translations) != len(english) {
			t.Fatalf("locale %s has %d keys, English has %d", currentLocale, len(translations), len(english))
		}
		for key := range english {
			if _, ok := translations[key]; !ok {
				t.Fatalf("locale %s is missing key %q", currentLocale, key)
			}
		}
	}
}

func TestFrontendTestsAreNotPublicAssets(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/static/ws-client.test.mjs", nil)
	responseRecorder := httptest.NewRecorder()

	newHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusNotFound {
		t.Fatalf("frontend test asset status code = %d, want %d", responseRecorder.Code, http.StatusNotFound)
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
