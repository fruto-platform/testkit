package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

const (
	maxJSONBodyBytes    = 64 * 1024
	maxGraphQLBodyBytes = 16 * 1024
	maxWebSocketBytes   = 4 * 1024
	defaultSSEInterval  = time.Second
	shutdownTimeout     = 5 * time.Second
	writeWait           = 10 * time.Second
	pongWait            = 60 * time.Second
	pingPeriod          = (pongWait * 9) / 10
)

// webFiles is embedded so the final scratch image needs no filesystem asset.
//
//go:embed templates/*.html templates/components/*.html static/*
var webFiles embed.FS

var pageTemplates = template.Must(
	template.New("base.html").
		Option("missingkey=error").
		ParseFS(webFiles, "templates/*.html", "templates/components/*.html"),
)

var version = "devel"

type response struct {
	Status         string `json:"status"`
	Version        string `json:"version"`
	UpstreamStatus int    `json:"upstreamStatus,omitempty"`
	UpstreamBody   string `json:"upstreamBody,omitempty"`
}

type item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type pageData struct {
	Title     string
	Version   string
	Protocols []protocolCard
	Clients   []webSocketClientView
}

type protocolCard struct {
	Index       string
	ID          string
	Name        string
	Endpoint    string
	Description string
	Status      string
}

type webSocketClientView struct {
	ID       string
	Label    string
	Endpoint string
}

type handlerConfig struct {
	sseInterval time.Duration
}

type application struct {
	handler http.Handler
	hub     *webSocketHub
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "probe" {
			os.Exit(runProbe(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
		}
		fmt.Fprintln(os.Stderr, "usage: testkit [probe URL]")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	app := newApplication(handlerConfig{sseInterval: sseIntervalFromEnv()})
	log.Printf("testkit %s listening on %s", version, listener.Addr())
	if err := serve(ctx, listener, app); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context, listener net.Listener, app *application) error {
	server := &http.Server{
		Handler:           app.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       30 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(listener)
	}()

	select {
	case err := <-serveResult:
		app.close()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		app.close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := <-serveResult; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func newHandler() http.Handler {
	return newHandlerWithConfig(handlerConfig{sseInterval: sseIntervalFromEnv()})
}

func newHandlerWithConfig(config handlerConfig) http.Handler {
	return newApplication(config).handler
}

func newApplication(config handlerConfig) *application {
	if config.sseInterval <= 0 {
		config.sseInterval = defaultSSEInterval
	}

	webSocketHub := newWebSocketHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/", staticIndex)
	mux.HandleFunc("/static/", staticAsset)
	mux.HandleFunc("/healthz", exactGET("/healthz", writeOK))
	mux.HandleFunc("/readyz", exactGET("/readyz", writeOK))
	mux.HandleFunc("/not-ready", exactGET("/not-ready", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusServiceUnavailable, response{Status: "not-ready", Version: version})
	}))
	mux.HandleFunc("/api/status", exactGET("/api/status", writeOK))
	mux.HandleFunc("/api/items", exactGET("/api/items", listItems))
	mux.HandleFunc("/api/echo", apiEcho)
	mux.HandleFunc("/graphql", graphQL)
	mux.HandleFunc("/events", exactGET("/events", func(writer http.ResponseWriter, request *http.Request) {
		events(writer, request, config.sseInterval)
	}))
	mux.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		webSocket(writer, request, webSocketHub)
	})

	return &application{handler: mux, hub: webSocketHub}
}

func (app *application) close() {
	app.hub.close()
}

func exactGET(path string, handler http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != path {
			http.NotFound(writer, request)
			return
		}
		if request.Method != http.MethodGet {
			writer.Header().Set("Allow", http.MethodGet)
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(writer, request)
	}
}

func staticIndex(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var page bytes.Buffer
	err := pageTemplates.ExecuteTemplate(&page, "base", pageData{
		Title:   "Transport Console",
		Version: version,
		Protocols: []protocolCard{
			{Index: "01", ID: "rest", Name: "REST", Endpoint: "/api/*", Description: "HTTP request and response checks", Status: "planned"},
			{Index: "02", ID: "graphql", Name: "GraphQL", Endpoint: "/graphql", Description: "Query, variables and error checks", Status: "planned"},
			{Index: "03", ID: "sse", Name: "Server-Sent Events", Endpoint: "/events", Description: "Streaming and reconnect checks", Status: "planned"},
		},
		Clients: []webSocketClientView{
			{ID: "client-a", Label: "Client A", Endpoint: "/ws"},
			{ID: "client-b", Label: "Client B", Endpoint: "/ws"},
		},
	})
	if err != nil {
		log.Printf("render dashboard: %v", err)
		http.Error(writer, "static page unavailable", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = page.WriteTo(writer)
}

func staticAsset(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	assetPath := strings.TrimPrefix(request.URL.Path, "/static/")
	if assetPath == "" || strings.Contains(assetPath, "..") {
		http.NotFound(writer, request)
		return
	}
	data, err := webFiles.ReadFile("static/" + assetPath)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	http.ServeContent(writer, request, assetPath, time.Time{}, bytes.NewReader(data))
}

func writeOK(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, response{Status: "ok", Version: version})
}

func listItems(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, []item{
		{ID: "item-1", Name: "First item"},
		{ID: "item-2", Name: "Second item"},
	})
}

func apiEcho(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/api/echo" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload json.RawMessage
	if err := decodeJSONBody(writer, request, maxJSONBodyBytes, &payload); err != nil {
		return
	}
	writeJSON(writer, http.StatusOK, struct {
		Data   json.RawMessage `json:"data"`
		Method string          `json:"method"`
		Path   string          `json:"path"`
	}{Data: payload, Method: request.Method, Path: request.URL.Path})
}

func graphQL(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Query         string                 `json:"query"`
		Variables     map[string]interface{} `json:"variables"`
		OperationName string                 `json:"operationName"`
	}
	if err := decodeJSONBody(writer, request, maxGraphQLBodyBytes, &payload); err != nil {
		return
	}
	if strings.TrimSpace(payload.Query) == "" {
		http.Error(writer, "GraphQL query is required", http.StatusBadRequest)
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         newGraphQLSchema(),
		RequestString:  payload.Query,
		VariableValues: payload.Variables,
		OperationName:  payload.OperationName,
	})
	writeJSON(writer, http.StatusOK, result)
}

func newGraphQLSchema() graphql.Schema {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"status": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.String),
				Resolve: func(graphql.ResolveParams) (interface{}, error) { return "ok", nil },
			},
			"version": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.String),
				Resolve: func(graphql.ResolveParams) (interface{}, error) { return version, nil },
			},
			"echo": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Args: graphql.FieldConfigArgument{
					"message": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					message, ok := params.Args["message"].(string)
					if !ok {
						return nil, errors.New("message must be a string")
					}
					return message, nil
				},
			},
		},
	})
	schema, err := graphql.NewSchema(graphql.SchemaConfig{Query: query})
	if err != nil {
		panic(err)
	}
	return schema
}

func events(writer http.ResponseWriter, request *http.Request, interval time.Duration) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for sequence := 1; ; sequence++ {
		data, err := json.Marshal(response{Status: "ok", Version: version})
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(writer, "id: %d\nevent: status\ndata: %s\n\n", sequence, data); err != nil {
			return
		}
		flusher.Flush()
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

type wsMessage struct {
	Message string `json:"message"`
	Version string `json:"version"`
}

type wsClient struct {
	hub  *webSocketHub
	conn *websocket.Conn
	send chan []byte
}

type webSocketHub struct {
	mutex   sync.Mutex
	clients map[*wsClient]struct{}
	closed  bool
}

func newWebSocketHub() *webSocketHub {
	return &webSocketHub{clients: make(map[*wsClient]struct{})}
}

func (hub *webSocketHub) register(client *wsClient) bool {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if hub.closed {
		return false
	}
	hub.clients[client] = struct{}{}
	return true
}

func (hub *webSocketHub) unregister(client *wsClient) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if _, ok := hub.clients[client]; ok {
		delete(hub.clients, client)
		close(client.send)
	}
}

func (hub *webSocketHub) send(message []byte) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	for client := range hub.clients {
		select {
		case client.send <- message:
		default:
			delete(hub.clients, client)
			close(client.send)
			_ = client.conn.Close()
		}
	}
}

func (hub *webSocketHub) close() {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()
	if hub.closed {
		return
	}
	hub.closed = true
	for client := range hub.clients {
		delete(hub.clients, client)
		close(client.send)
		_ = client.conn.Close()
	}
}

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     sameWebSocketOrigin,
}

func sameWebSocketOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil || (parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https") {
		return false
	}
	return strings.EqualFold(parsedOrigin.Host, request.Host)
}

func webSocket(writer http.ResponseWriter, request *http.Request, hub *webSocketHub) {
	connection, err := webSocketUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	client := &wsClient{hub: hub, conn: connection, send: make(chan []byte, 16)}
	if !client.hub.register(client) {
		_ = connection.Close()
		return
	}
	defer func() {
		client.hub.unregister(client)
		_ = connection.Close()
	}()

	go client.writePump()
	client.readPump()
}

func (client *wsClient) readPump() {
	client.conn.SetReadLimit(maxWebSocketBytes)
	_ = client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		var payload struct {
			Message string `json:"message"`
		}
		if err := client.conn.ReadJSON(&payload); err != nil {
			return
		}
		message, err := json.Marshal(wsMessage{Message: payload.Message, Version: version})
		if err != nil {
			return
		}
		client.hub.send(message)
	}
}

func (client *wsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func decodeJSONBody(writer http.ResponseWriter, request *http.Request, limit int64, destination interface{}) error {
	request.Body = http.MaxBytesReader(writer, request.Body, limit)
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(destination); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			http.Error(writer, "request body too large", http.StatusRequestEntityTooLarge)
			return err
		}
		http.Error(writer, "invalid JSON request", http.StatusBadRequest)
		return err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			http.Error(writer, "request body must contain one JSON value", http.StatusBadRequest)
		} else {
			http.Error(writer, "invalid JSON request", http.StatusBadRequest)
		}
		return err
	}
	return nil
}

func sseIntervalFromEnv() time.Duration {
	interval, err := time.ParseDuration(os.Getenv("SSE_INTERVAL"))
	if err != nil || interval <= 0 {
		return defaultSSEInterval
	}
	return interval
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}
