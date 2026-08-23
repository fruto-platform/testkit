# Fruto Testkit

Imagem Go autocontida para validar clientes e páginas estáticas que consomem REST, GraphQL, Server-Sent Events (SSE) e WebSocket.

## Executar

```sh
docker build --tag fruto-testkit .
docker run --rm --publish 8080:8080 fruto-testkit
```

A página estática fica disponível em <http://localhost:8080/>. A imagem executa como UID/GID `65532:65532`.

O intervalo dos eventos SSE pode ser alterado sem rebuild:

```sh
docker run --rm --publish 8080:8080 --env SSE_INTERVAL=250ms fruto-testkit
```

## Endpoints

| Endpoint | Uso |
| --- | --- |
| `GET /` | Página HTML estática embutida |
| `GET /static/style.css` | Asset estático embutido |
| `GET /healthz` | Liveness determinístico |
| `GET /readyz` | Readiness determinístico |
| `GET /api/status` | REST status/version |
| `GET /api/items` | Lista REST determinística |
| `POST /api/echo` | Echo de um payload JSON |
| `POST /graphql` | Query GraphQL |
| `GET /events` | Stream SSE persistente |
| `GET /ws` | WebSocket JSON com echo e broadcast |
| `GET /outbound?url=...` | GET controlado a um upstream HTTP/HTTPS |

O endpoint `/not-ready` permanece disponível para testes de status `503`.

## Exemplos

REST:

```sh
curl http://localhost:8080/api/status
curl http://localhost:8080/api/items
curl --json '{"hello":"world"}' http://localhost:8080/api/echo
```

GraphQL:

```sh
curl --json '{"query":"{ status version echo(message: \\"hello\\") }"}' \
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

Cada cliente WebSocket conectado recebe a mensagem no formato:

```json
{"message":"hello","version":"devel"}
```
