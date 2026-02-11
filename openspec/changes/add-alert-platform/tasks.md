# Implementation Tasks: Alert Aggregation Platform

## 1. Foundation

- [x] 1.1 Create project skeleton: backend (Golang), frontend (React + Ant Design), Docker Compose for dev.
- [x] 1.2 Define unified alert data model and storage schema (alert_id, source_id, labels, status, etc.).
- [x] 1.3 Implement auth API (login, logout, token/session) and protect routes; minimal user store.

## 2. Data ingestion and configuration

- [x] 2.1 Implement datasource CRUD API and webhook receivers (Prometheus, VictoriaMetrics, Elasticsearch, Doris); normalize incoming alerts to unified model.
- [x] 2.2 Implement channel CRUD API and senders (Telegram, Lark); secure storage for secrets; test-send.
- [x] 2.3 Implement template CRUD API and tag-based rendering; preview with sample data.

## 3. Rules engine

- [x] 3.1 Implement rule CRUD API, enable/disable, match conditions, routing to channels and templates.
- [x] 3.2 Add check frequency, duration threshold, and exclude time windows per rule.
- [x] 3.3 Add alert recovery notification and send-rate limit (e.g. per-alert interval).
- [x] 3.4 Add same-type aggregation by hostname / host IP / port and smart send (one notification per group).
- [x] 3.5 Add tag-based suppression (source condition, suppressed condition, duration).
- [x] 3.6 Add Jira integration: create ticket when alert fires N times; store Jira key on alert/history.
- [x] 3.7 Rule JSON import/export and batch enable/disable/delete.

## 4. History and reporting

- [x] 4.1 Persist every ingested alert with unique alert_id; expose alert-history query API (filters, pagination, detail).
- [x] 4.2 Reports API: aggregate by time, datasource, severity, tags; return data for charts.
- [x] 4.3 Export alerts/reports to CSV or Excel; enforce auth.

## 5. Frontend and validation

- [x] 5.1 Login UI and token handling; guard config/history/reports pages.
- [x] 5.2 Datasource and channel config UIs; template editor with preview.
- [x] 5.3 Rules UI: list, detail, enable/disable, import/export, batch actions; suppression and Jira config.
- [x] 5.4 Alert history list and detail; reports views and export.
- [x] 5.5 Automated tests for critical paths (ingest → rule → channel, history query, export).
