# Fruto Testkit

[English](../../README.md) |
[Español (Argentina)](../es-AR/README.md)

> Projeto experimental em pré-release. O Fruto Testkit é uma fixture de testes,
> não uma aplicação de produção.

O Fruto Testkit é uma pequena imagem de contêiner determinística para exercitar
aplicações HTTP, entrega de aplicações no Kubernetes e políticas de rede. Um único
binário Go estático fornece REST, GraphQL, Server-Sent Events (SSE), WebSocket,
endpoints de saúde e um comando explícito de probe de saída.

O servidor HTTP público e o probe de rede são intencionalmente separados. Executar
o servidor não transforma uma carga de teste exposta publicamente em um proxy de
saída; diagnósticos de rede exigem um comando explícito do contêiner.

## Capacidades

| Capacidade | Interface |
| --- | --- |
| Alias de detecção de idioma | `GET /`, `GET /websocket` |
| Páginas localizadas | `GET /en/`, `GET /pt-BR/`, `GET /es-AR/` |
| Console WebSocket localizado | `GET /en/websocket`, `GET /pt-BR/websocket`, `GET /es-AR/websocket` |
| Asset estático | `GET /static/style.css` |
| Liveness | `GET /healthz` |
| Readiness | `GET /readyz` |
| Resposta indisponível | `GET /not-ready` |
| Status REST | `GET /api/status` |
| Coleção REST | `GET /api/items` |
| Echo REST | `POST /api/echo` |
| GraphQL | `POST /graphql` |
| Server-Sent Events | `GET /events` |
| Echo e broadcast WebSocket | `GET /ws` |
| Probe de rede HTTP/HTTPS | `testkit probe URL` |

As respostas incluem a versão de build injetada pelo argumento Docker `VERSION`.
Isso torna testes de rollout e transporte observáveis sem alterar a identidade
lógica da carga.

## Pré-requisitos

- Go 1.26.x.
- Docker Engine ou Docker Desktop com Buildx habilitado.
- `curl` para os exemplos.

## Build e execução local

Construa a imagem para a plataforma local do Docker:

```sh
docker buildx build \
  --load \
  --build-arg VERSION=dev \
  --tag fruto-testkit:dev \
  .
```

Execute-a com o contrato restrito esperado pela Fruto Platform:

```sh
docker run --rm \
  --publish 8080:8080 \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  fruto-testkit:dev
```

Valide o contrato REST:

```sh
curl --fail http://localhost:8080/api/status
```

Resposta esperada:

```json
{"status":"ok","version":"dev"}
```

## Console no navegador

Abra `http://localhost:8080/` para detectar o idioma do navegador ou use
diretamente `/en/`, `/pt-BR/` ou `/es-AR/`. O laboratório WebSocket fica no
caminho `/websocket` correspondente. Ele mostra dois
clientes independentes da mesma origem, `Client A` e `Client B`, permitindo
conectar os dois painéis, enviar uma mensagem por um deles e alternar entre os
eventos JSON brutos e uma visão de chat com horários locais de envio e
recebimento. As conexões são explícitas: a interface não reconecta
automaticamente após erro ou encerramento.

## Exemplos de protocolos

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

Cada cliente WebSocket conectado recebe a mensagem de broadcast com a versão de
build atual:

```json
{"message":"hello","version":"dev"}
```

## Probe de rede

Execute diagnósticos de rede como um comando explícito da mesma imagem:

```sh
docker run --rm fruto-testkit:dev probe https://example.com/
```

O probe executa uma única requisição HTTP ou HTTPS `GET`, com limites, e emite um
resultado JSON. Seus códigos de saída formam o contrato de automação:

| Código | Significado |
| --- | --- |
| `0` | O destino retornou HTTP `2xx`. |
| `1` | A requisição falhou ou retornou uma resposta diferente de `2xx`. |
| `2` | Os argumentos do comando ou a URL são inválidos. |

Para validar NetworkPolicies do Kubernetes, execute o probe como um Job de curta
duração no namespace testado. Aplique as labels e a ServiceAccount cuja identidade
de rede deseja validar e verifique o código de saída do Job. Assim, origem, destino
e resultado esperado de permissão ou negação permanecem explícitos.

## Contrato do contêiner

- Escuta na porta TCP `8080`.
- Executa como UID/GID `65532:65532`.
- Suporta filesystem raiz somente leitura.
- Não exige capabilities Linux nem elevação de privilégios.
- Inclui CA bundle para probes HTTPS.
- Incorpora todos os assets estáticos no binário.
- Compila de forma reproduzível para plataformas alvo do BuildKit, incluindo
  `linux/amd64` e `linux/arm64`.
- Trata `SIGTERM` e drena conexões HTTP, SSE e WebSocket com shutdown limitado.

O servidor aceita conexões WebSocket de mesma origem e clientes que omitem o
header `Origin`, como ferramentas de linha de comando. Conexões cross-origin de
navegadores são rejeitadas.

## Configuração

| Variável | Padrão | Finalidade |
| --- | --- | --- |
| `SSE_INTERVAL` | `1s` | Intervalo entre eventos de status SSE. |

Valores inválidos ou não positivos de `SSE_INTERVAL` usam o padrão.

## Distribuição da imagem

Releases versionadas publicam imagens multiplataforma no GitHub Container
Registry:

```text
ghcr.io/fruto-platform/testkit:v0.1.1
```

As tags existem para descoberta. Testes automatizados devem consumir o digest
imutável informado pela pipeline de release:

```text
ghcr.io/fruto-platform/testkit@sha256:<digest>
```

O projeto não publica uma tag `latest`. Assinatura, SBOMs e attestations adicionais
de proveniência estão fora do contrato atual de release.

## Desenvolvimento

Execute o gate local de qualidade:

```sh
gofmt -w *.go
go test -race -cover ./...
go vet ./...
go mod tidy -diff
```

Construa e exercite o contêiner final sempre que runtime, assets incorporados,
probes ou Dockerfile forem alterados.

## Documentação

Inglês é o idioma canônico da documentação. Traduções disponíveis:

- [English](../../README.md)
- [Español (Argentina)](../es-AR/README.md)

As traduções preservam comandos, caminhos, endpoints, campos e identificadores de
protocolo em inglês. Em caso de divergência, a versão em inglês define o contrato
atual.

## Contribuindo

Leia [CONTRIBUTING.md](CONTRIBUTING.md) antes de propor uma alteração.

## Segurança

Não reporte vulnerabilidades em issues públicas. Siga [SECURITY.md](SECURITY.md).

## Licença

Licenciado sob a [Apache License 2.0](../../LICENSE).
