# Molejo Testkit

[Português (Brasil)](docs/pt-BR/README.md) |
[Español (Argentina)](docs/es-AR/README.md)

> Experimental pre-release project. Molejo Testkit is a test fixture, not a
> production application.

Molejo Testkit is a small, deterministic container image for exercising HTTP
applications, Kubernetes application delivery, and network policies. A single
static Go binary provides REST, GraphQL, Server-Sent Events (SSE), WebSocket,
health endpoints, and an explicit outbound probe command.

The public HTTP server and the network probe are intentionally separate. Running
the server cannot turn a publicly exposed test workload into an outbound proxy;
network diagnostics require an explicit container command.

## Capabilities

| Capability | Interface |
| --- | --- |
| Language detection aliases | `GET /`, `GET /websocket`, `GET /rest`, `GET /graphql-lab`, `GET /sse` |
| Localized browser pages | `GET /en/`, `GET /pt-BR/`, `GET /es-AR/` |
| Localized browser labs | `GET /en/rest`, `/en/graphql-lab`, `/en/sse`, `/en/websocket` and matching translated paths |
| Static asset | `GET /static/style.css` |
| Liveness | `GET /healthz` |
| Readiness | `GET /readyz` |
| Unready response | `GET /not-ready` |
| REST status | `GET /api/status` |
| REST collection | `GET /api/items` |
| REST echo | `POST /api/echo` |
| GraphQL | `POST /graphql` |
| Server-Sent Events | `GET /events` |
| WebSocket echo and broadcast | `GET /ws` |
| HTTP/HTTPS network probe | `testkit probe URL` |

Responses include the build version injected through the Docker `VERSION` build
argument. This makes rollout and transport tests observable without changing the
logical identity of the workload.

## Prerequisites

- Go 1.26.x.
- Docker Engine or Docker Desktop with Buildx enabled.
- `curl` for the examples.

## Build and run locally

Build the image for the local Docker platform:

```sh
docker buildx build \
  --load \
  --build-arg VERSION=dev \
  --tag molejo-testkit:dev \
  .
```

Run it using the restricted container contract expected by Molejo Platform:

```sh
docker run --rm \
  --publish 8080:8080 \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  molejo-testkit:dev
```

Verify the REST contract:

```sh
curl --fail http://localhost:8080/api/status
```

Expected response:

```json
{"status":"ok","version":"dev"}
```

## Browser console

Open `http://localhost:8080/` to let the server detect the browser language, or
use a localized URL directly: `/en/`, `/pt-BR/` or `/es-AR/`. The browser labs
are available at the matching `/rest`, `/graphql-lab`, `/sse` and `/websocket`
paths. REST and GraphQL use guided presets that render request and response
details. The SSE lab displays event IDs, names and data, with explicit connect,
disconnect and reconnect actions. The WebSocket lab shows two independent
same-origin clients, `Client A` and `Client B`, so you can connect both panels,
send a message from either one, and switch between raw JSON events and a chat
view with local send/receive timestamps. Browser clients do not reconnect
automatically after an error or close.

## Protocol examples

REST echo:

```sh
curl --json '{"hello":"world"}' http://localhost:8080/api/echo
```

GraphQL:

```sh
curl --json '{"query":"{ status version echo(message: \"hello\") }"}' \
  http://localhost:8080/graphql
```

SSE:

```sh
curl -N http://localhost:8080/events
```

WebSocket:

```sh
wscat --connect ws://localhost:8080/ws
> {"message":"hello"}
```

Each connected WebSocket client receives the broadcast message with the current
build version:

```json
{"message":"hello","version":"dev"}
```

## Network probe

Run network diagnostics as an explicit command of the same image:

```sh
docker run --rm molejo-testkit:dev probe https://example.com/
```

The probe performs one bounded HTTP or HTTPS `GET` request and emits a JSON result.
Its exit codes form the automation contract:

| Exit code | Meaning |
| --- | --- |
| `0` | The destination returned HTTP `2xx`. |
| `1` | The request failed or returned a non-`2xx` response. |
| `2` | The command arguments or URL are invalid. |

To verify Kubernetes NetworkPolicies, run the probe as a short-lived Job in the
namespace under test. Apply the labels and ServiceAccount whose network identity
you want to validate, then assert the Job exit code. This keeps the source,
destination, and expected allow-or-deny result explicit.

## Container contract

- Listens on TCP port `8080`.
- Runs as UID/GID `65532:65532`.
- Supports a read-only root filesystem.
- Requires no Linux capabilities or privilege escalation.
- Includes a CA bundle for HTTPS probes.
- Embeds all static assets in the binary.
- Builds reproducibly for BuildKit target platforms, including `linux/amd64` and
  `linux/arm64`.
- Handles `SIGTERM` and drains HTTP, SSE, and WebSocket connections with a bounded
  shutdown.

The HTTP server accepts same-origin WebSocket connections and clients that omit
the `Origin` header, such as command-line test clients. Cross-origin browser
connections are rejected.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `SSE_INTERVAL` | `1s` | Interval between SSE status events. |

Invalid or non-positive `SSE_INTERVAL` values fall back to the default.

## Structured logs

Server mode writes newline-delimited JSON logs to standard output. Every record
includes `service` and `version`. The primary `event` values are:

- `server.started`, `server.stopped`, and server failure events;
- `http.request.completed` for `/api/status`, `/api/items`, and `/api/echo`, with
  method, stable route, status code, duration in milliseconds, and response bytes;
- `connection.opened` and `connection.closed` for WebSocket and SSE, with the
  active protocol and total connection counts and connection duration on close;
- `connections.snapshot` every 15 minutes while at least one connection is active.

Connection lifecycle events and snapshots include `connection_sequence`, which
increases with every connection state transition in the process. Consumers can
use it to reconstruct transition order when concurrent log records arrive out of
order.

Completed REST requests and accepted WebSocket and SSE connections include a
UUIDv7 `correlation_id`. REST clients may provide it through
`X-Testkit-Correlation-ID`; WebSocket and SSE clients may use the
`correlation_id` query parameter. Missing or invalid values are replaced with a
server-generated ID. REST responses return the effective ID in the same header.
The bundled REST, WebSocket, and SSE labs generate and display these IDs
automatically. Connection snapshots remain aggregate and do not include
correlation IDs.

Connection counts represent accepted connections currently observed by one
server process and reset on restart. Page refreshes, normal closes, and transport
errors decrement the count when the server observes the disconnect. WebSocket
heartbeats bound silent failure detection to about 60 seconds; SSE detection is
best effort when a network path disappears without closing the HTTP stream.
Graceful shutdown waits for accepted WebSocket handlers to emit their closing
facts. A crash or forced process termination cannot emit closing facts, so
consumers must treat each server start as a new process-local epoch.

Logs do not include client addresses, raw headers, raw query strings, request
payloads, WebSocket messages, or SSE data. They are observable test facts, not
durable or global metrics.

## Image distribution

Tagged releases publish multi-platform images to GitHub Container Registry:

```text
ghcr.io/molejo-platform/testkit:v0.3.0
```

Tags are provided for discovery. Automated tests should consume the immutable
digest reported by the release workflow:

```text
ghcr.io/molejo-platform/testkit@sha256:<digest>
```

The project does not publish a `latest` tag. Signing, SBOMs, and additional
provenance attestations are outside the current release contract.

## Development

Run the local quality gate:

```sh
gofmt -w *.go
go test -race -cover ./...
go vet ./...
go mod tidy -diff
node --test frontend-tests/ws-client.test.mjs frontend-tests/ws-ui.test.mjs
```

Build and exercise the final container whenever runtime, embedded assets, probes,
or the Dockerfile changes.

## Documentation

English is the canonical documentation language. Available translations:

- [Português (Brasil)](docs/pt-BR/README.md)
- [Español (Argentina)](docs/es-AR/README.md)

Translations preserve commands, paths, endpoint names, fields, and protocol
identifiers in English. If translated content diverges, the English version
defines the current contract.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) before proposing a change.

## Security

Do not report vulnerabilities through public issues. Follow
[SECURITY.md](SECURITY.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
