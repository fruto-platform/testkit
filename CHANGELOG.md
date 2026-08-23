# Changelog

All notable changes to Fruto Testkit are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

No changes yet.

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

[Unreleased]: https://github.com/fruto-platform/testkit/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/fruto-platform/testkit/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/fruto-platform/testkit/compare/v0.0.2...v0.1.0
[0.0.2]: https://github.com/fruto-platform/testkit/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/fruto-platform/testkit/releases/tag/v0.0.1
