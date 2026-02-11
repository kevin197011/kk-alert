# Change: Add Alert Aggregation Platform (split PRD into OpenSpec specs)

## Why

The product requirements document (`docs/requirements-alert-platform.md`) defines the full scope of the alert aggregation platform. Splitting it into OpenSpec capabilities provides traceable, testable requirements per domain (auth, datasources, channels, templates, rules, alert-history, reports) and a clear implementation checklist.

## What Changes

- Introduce seven capability specs as **ADDED** requirements, mapped from the PRD:
  - **auth**: Login, session (JWT/session), logout, optional user management.
  - **datasources**: Multi-source alert ingestion (Prometheus, VictoriaMetrics, ES, Doris), CRUD, test connection, normalization to unified model.
  - **channels**: Notification channels (Telegram, Lark), CRUD, secure storage, test send.
  - **templates**: Tag-based alert templates, placeholders, preview.
  - **rules**: Alert rules (match, route, enable/disable, frequency, duration, time windows, recovery, rate limit, aggregation by hostname/IP/port, Jira on N occurrences, tag-based suppression), JSON import/export, batch operations.
  - **alert-history**: Unique alert ID, storage, query by multiple dimensions, detail view.
  - **reports**: Aggregation dimensions, charts, CSV/Excel export, access control.
- No existing specs or code are modified; this is greenfield.

## Impact

- Affected specs: auth, datasources, channels, templates, rules, alert-history, reports (all new).
- Affected code: none yet; implementation will follow `tasks.md` after approval.
- Reference: `docs/requirements-alert-platform.md` (v0.9).
