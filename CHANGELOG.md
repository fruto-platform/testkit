# Changelog

All notable changes to Molejo Testkit are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

No changes yet.

## [0.4.0] - 2026-08-28

### Added

- UUIDv7 correlation IDs for completed REST requests and accepted WebSocket and
  SSE connections, with validated client propagation, safe server fallback, and
  automatic generation and display in the bundled browser labs.

### Changed

- Clarified that active connection facts are process-local, how client
  disconnects are detected, and the limits of silent SSE failure detection.

### Fixed

- Graceful shutdown now waits for accepted WebSocket handlers to emit their
  closing facts before the server exits.

## [0.3.0] - 2026-08-28

### Added

- Structured JSON logs for server lifecycle, REST request completion, and
  WebSocket and SSE connection lifecycle.
- Per-process active connection snapshots every 15 minutes while at least one
  WebSocket or SSE connection is active.
- Monotonic per-process connection sequences so concurrent lifecycle facts can
  be reconstructed in transition order.

## [0.2.0] - 2026-08-24

### Added

- Localized REST, GraphQL, and Server-Sent Events browser laboratories.
- Guided REST presets for status, items, echo, and error responses.
- Guided GraphQL presets for status/version, variables, and invalid queries.
- Explicit SSE connect, disconnect, and reconnect controls with event details.
- Browser regression coverage for the new laboratories and localized routes.

### Changed

- The home protocol map now links to active REST, GraphQL, SSE, and WebSocket
  laboratories.
- README documentation now includes the localized browser laboratory aliases.

## [0.1.1] - 2026-08-23

### Fixed

- Updated the release pipeline to use the Node.js 24-compatible `setup-node` action runtime.

## [0.1.0] - 2026-08-23

### Added

- Localized browser pages for English, Brazilian Portuguese, and Argentine Spanish.
- Automatic locale detection using query parameter, cookie, and `Accept-Language`.
- JSON translation catalogs embedded in the Go binary.
- Localized WebSocket dashboard routes with language switcher and breadcrumbs.
- Chat and JSON WebSocket views with localized relative timestamps.
- Frontend test execution in the release workflow.

### Changed

- `/` and `/websocket` now redirect to the detected localized route.
- WebSocket broadcasts with identical payloads remain visible as separate received events.
- Browser interface text is centralized in locale catalogs, including the brand name.

## [0.0.2] - 2026-08-23

### Added

- Browser WebSocket console with two independent clients.
- Dedicated WebSocket page with JSON and chat-oriented event views.
- Local send and receive timestamps for browser-side WebSocket events.

## [0.0.1] - 2026-08-23

### Added

- Initial multi-platform image publication workflow for GitHub Container Registry.

[Unreleased]: https://github.com/molejo-platform/testkit/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/molejo-platform/testkit/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/molejo-platform/testkit/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/molejo-platform/testkit/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/molejo-platform/testkit/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/molejo-platform/testkit/compare/v0.0.2...v0.1.0
[0.0.2]: https://github.com/molejo-platform/testkit/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/molejo-platform/testkit/releases/tag/v0.0.1
