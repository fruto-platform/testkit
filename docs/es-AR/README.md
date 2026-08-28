# Molejo Testkit

[English](../../README.md) |
[Português (Brasil)](../pt-BR/README.md)

> Proyecto experimental en pre-release. Molejo Testkit es una fixture de pruebas,
> no una aplicación de producción.

Molejo Testkit es una imagen de contenedor pequeña y determinista para ejercitar
aplicaciones HTTP, entrega de aplicaciones en Kubernetes y políticas de red. Un
único binario Go estático ofrece REST, GraphQL, Server-Sent Events (SSE),
WebSocket, endpoints de salud y un comando explícito de probe de egreso.

El servidor HTTP público y el probe de red están separados intencionalmente.
Ejecutar el servidor no convierte una carga de prueba expuesta públicamente en un
proxy de egreso; los diagnósticos de red requieren un comando explícito del
contenedor.

## Capacidades

| Capacidad | Interfaz |
| --- | --- |
| Aliases de detección de idioma | `GET /`, `GET /websocket`, `GET /rest`, `GET /graphql-lab`, `GET /sse` |
| Páginas localizadas | `GET /en/`, `GET /pt-BR/`, `GET /es-AR/` |
| Laboratorios localizados en el navegador | `GET /en/rest`, `/en/graphql-lab`, `/en/sse`, `/en/websocket` y caminos traducidos equivalentes |
| Asset estático | `GET /static/style.css` |
| Liveness | `GET /healthz` |
| Readiness | `GET /readyz` |
| Respuesta no disponible | `GET /not-ready` |
| Estado REST | `GET /api/status` |
| Colección REST | `GET /api/items` |
| Echo REST | `POST /api/echo` |
| GraphQL | `POST /graphql` |
| Server-Sent Events | `GET /events` |
| Echo y broadcast WebSocket | `GET /ws` |
| Probe de red HTTP/HTTPS | `testkit probe URL` |

Las respuestas incluyen la versión de build inyectada mediante el argumento
Docker `VERSION`. Esto hace observables las pruebas de rollout y transporte sin
cambiar la identidad lógica de la carga.

## Requisitos previos

- Go 1.26.x.
- Docker Engine o Docker Desktop con Buildx habilitado.
- `curl` para los ejemplos.

## Build y ejecución local

Construí la imagen para la plataforma local de Docker:

```sh
docker buildx build \
  --load \
  --build-arg VERSION=dev \
  --tag molejo-testkit:dev \
  .
```

Ejecutala con el contrato restringido esperado por Molejo Platform:

```sh
docker run --rm \
  --publish 8080:8080 \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  molejo-testkit:dev
```

Validá el contrato REST:

```sh
curl --fail http://localhost:8080/api/status
```

Respuesta esperada:

```json
{"status":"ok","version":"dev"}
```

## Consola del navegador

Abra `http://localhost:8080/` para detectar el idioma del navegador o use
directamente `/en/`, `/pt-BR/` o `/es-AR/`. Los laboratorios están disponibles
en los caminos `/rest`, `/graphql-lab`, `/sse` y `/websocket` correspondientes.
REST y GraphQL usan presets guiados que muestran detalles de solicitud y
respuesta. El laboratorio SSE muestra IDs, nombres y datos de eventos, con
acciones explícitas para conectar, desconectar y reconectar. El laboratorio
WebSocket muestra dos clientes independientes del mismo origen, `Client A` y
`Client B`, para conectar ambos paneles, enviar un mensaje desde cualquiera y
alternar entre eventos JSON crudos y una vista de chat con horarios locales de
envío y recepción. Los clientes del navegador no se reconectan automáticamente
después de un error o cierre.

## Ejemplos de protocolos

Echo REST:

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

Cada cliente WebSocket conectado recibe el mensaje de broadcast con la versión de
build actual:

```json
{"message":"hello","version":"dev"}
```

## Probe de red

Ejecutá diagnósticos de red como un comando explícito de la misma imagen:

```sh
docker run --rm molejo-testkit:dev probe https://example.com/
```

El probe realiza una única solicitud HTTP o HTTPS `GET`, con límites, y emite un
resultado JSON. Sus códigos de salida forman el contrato de automatización:

| Código | Significado |
| --- | --- |
| `0` | El destino respondió con HTTP `2xx`. |
| `1` | La solicitud falló o devolvió una respuesta distinta de `2xx`. |
| `2` | Los argumentos del comando o la URL son inválidos. |

Para validar NetworkPolicies de Kubernetes, ejecutá el probe como un Job de corta
duración en el namespace bajo prueba. Aplicá las labels y la ServiceAccount cuya
identidad de red querés validar y comprobá el código de salida del Job. Así, el
origen, el destino y el resultado esperado de permiso o denegación quedan
explícitos.

## Contrato del contenedor

- Escucha en el puerto TCP `8080`.
- Se ejecuta como UID/GID `65532:65532`.
- Admite un filesystem raíz de solo lectura.
- No requiere capabilities Linux ni escalamiento de privilegios.
- Incluye un CA bundle para probes HTTPS.
- Incorpora todos los assets estáticos en el binario.
- Se compila de manera reproducible para plataformas objetivo de BuildKit,
  incluyendo `linux/amd64` y `linux/arm64`.
- Maneja `SIGTERM` y drena conexiones HTTP, SSE y WebSocket con un shutdown
  limitado.

El servidor acepta conexiones WebSocket del mismo origen y clientes que omiten el
header `Origin`, como las herramientas de línea de comandos. Las conexiones
cross-origin de navegadores son rechazadas.

## Configuración

| Variable | Valor predeterminado | Finalidad |
| --- | --- | --- |
| `SSE_INTERVAL` | `1s` | Intervalo entre eventos de estado SSE. |

Los valores inválidos o no positivos de `SSE_INTERVAL` usan el valor
predeterminado.

## Logs estructurados

El modo servidor escribe logs JSON delimitados por línea en la salida estándar.
Cada registro incluye `service` y `version`. Los principales valores de `event`
son:

- `server.started`, `server.stopped` y eventos de falla del servidor;
- `http.request.completed` para `/api/status`, `/api/items` y `/api/echo`, con
  método, ruta estable, código de estado, duración en milisegundos y bytes de
  respuesta;
- `connection.opened` y `connection.closed` para WebSocket y SSE, con las
  cantidades de conexiones activas del protocolo y totales y la duración al cerrar;
- `connections.snapshot` cada 15 minutos mientras haya al menos una conexión
  activa.

Los eventos de ciclo de vida y snapshots incluyen `connection_sequence`, que
aumenta con cada transición de estado de conexión en el proceso. Los consumidores
pueden usarla para reconstruir el orden de las transiciones cuando los registros
concurrentes llegan fuera de orden.

Las cantidades de conexiones son locales a un proceso y se reinician junto con
él. Los logs no incluyen direcciones de clientes, headers, query strings,
payloads de requests, mensajes WebSocket ni datos SSE. Son hechos de prueba
observables, no métricas durables o globales.

## Distribución de la imagen

Las releases versionadas publican imágenes multiplataforma en GitHub Container
Registry:

```text
ghcr.io/molejo-platform/testkit:v0.1.1
```

Los tags existen para descubrimiento. Las pruebas automatizadas deben consumir el
digest inmutable informado por el pipeline de release:

```text
ghcr.io/molejo-platform/testkit@sha256:<digest>
```

El proyecto no publica un tag `latest`. La firma, los SBOMs y las attestations
adicionales de procedencia quedan fuera del contrato actual de release.

## Desarrollo

Ejecutá el gate local de calidad:

```sh
gofmt -w *.go
go test -race -cover ./...
go vet ./...
go mod tidy -diff
```

Construí y ejercitá el contenedor final siempre que cambien el runtime, los assets
incorporados, los probes o el Dockerfile.

## Documentación

El inglés es el idioma canónico de la documentación. Traducciones disponibles:

- [English](../../README.md)
- [Português (Brasil)](../pt-BR/README.md)

Las traducciones conservan en inglés los comandos, paths, endpoints, campos e
identificadores de protocolo. Si existe una divergencia, la versión en inglés
define el contrato vigente.

## Contribuir

Leé [CONTRIBUTING.md](CONTRIBUTING.md) antes de proponer un cambio.

## Seguridad

No reportes vulnerabilidades en issues públicas. Seguí
[SECURITY.md](SECURITY.md).

## Licencia

Licenciado bajo la [Apache License 2.0](../../LICENSE).
