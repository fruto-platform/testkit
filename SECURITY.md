# Security Policy

[Português (Brasil)](docs/pt-BR/SECURITY.md) |
[Español (Argentina)](docs/es-AR/SECURITY.md)

English is the canonical version of this policy.

## Supported versions

Fruto Testkit is an experimental pre-release project. It has no supported or
production-ready release. Security fixes are applied to the default branch on a
best-effort basis.

## Intended use

Testkit is a deterministic test fixture. It is not an authenticated application,
an authorization service, or a production backend. Deploy it only where a test
workload is appropriate and remove disposable environments after validation.

The public HTTP server intentionally does not expose the outbound network probe.
The probe is an explicit container command intended for controlled Jobs and local
diagnostics. Anyone able to execute that command receives the network identity of
the container or Pod, so access to workload creation and execution remains a
cluster security boundary.

## Reporting a vulnerability

GitHub private vulnerability reporting is not enabled for this repository yet.
Until a private reporting channel is available, contact the maintainers through
the [Fruto Platform organization profile](https://github.com/fruto-platform) to
request a private channel without including vulnerability details in the initial
message. Do not disclose the vulnerability in a public issue, discussion, pull
request, or test log.

Include, when possible:

- the affected revision and component;
- steps to reproduce or a minimal proof of concept;
- the expected security impact;
- relevant deployment assumptions;
- any known mitigation.

The maintainers will acknowledge and assess reports as capacity permits. Because
the project is pre-release, no response or remediation service-level agreement is
currently provided.

## Disclosure

Allow the maintainers a reasonable opportunity to investigate and prepare a fix
before public disclosure. Credit will be coordinated with the reporter when
appropriate and desired.
