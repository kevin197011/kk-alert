# Design: Alert Aggregation Platform

## Context

Platform aggregates alerts from multiple sources (Prometheus, VictoriaMetrics, ES, Doris), normalizes them, applies rules (routing, suppression, aggregation, Jira), and sends notifications via Telegram and Lark. Requirements are captured in `docs/requirements-alert-platform.md`. Backend is Golang; frontend is React + Ant Design; architecture is frontend-backend separation.

## Goals / Non-Goals

- Goals: Single place to receive, store, route, and report on alerts; configurable rules and templates; traceability via unique alert ID and history.
- Non-Goals: Replacing data sources’ own evaluation; full ITSM; arbitrary new channel types in v1 (only Telegram, Lark).

## Decisions

- **Unified alert model**: All inbound alerts are normalized to a common schema (alert_id, source_id, source_type, title, severity, status, firing_at/resolved_at, labels, annotations, raw). Rules and templates operate on this model.
- **Rule scope**: Each rule has match conditions, routing (channels + template), and optional: check frequency, duration, time windows, recovery, rate limit, aggregation by hostname/IP/port, suppression (tag-based), Jira on N occurrences. Rules are ordered; first match wins or all matches apply (implementation choice).
- **Aggregation by dimension**: “Same type” = same rule + same labels except the aggregation dimension (e.g. instance/host/ip/port). Within a time window, multiple alerts of the same type are merged into one notification with aggregated host/IP/port list (or count).
- **Suppression**: Tag-based: when an alert matches “source” condition, a time window starts during which alerts matching “suppressed” condition are not sent (still stored).
- **Sensitive config**: Channel tokens and Jira credentials encrypted at rest; API never returns plain secrets.
- **APIs**: RESTful; auth via JWT (or session). Suggested modules: /api/v1/auth, /api/v1/datasources, /api/v1/channels, /api/v1/templates, /api/v1/rules, /api/v1/alerts, /api/v1/reports; inbound webhooks under /api/v1/inbound/*.

## Risks / Trade-offs

- Rule complexity (many knobs) may lead to subtle bugs; mitigate with clear defaults and tests per scenario.
- High alert volume may require batching and async send; design for idempotent write and rate limiting.

## Migration Plan

N/A (greenfield). Rollout: deploy backend and frontend; configure datasources and channels; import or create rules.

## Open Questions

- Default “unmatched alert” policy: store only vs. default route (to be decided in implementation).
- Jira: one ticket per alert group vs. per alert when N reached (spec recommends one per alert group).
