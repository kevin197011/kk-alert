# Implementation Tasks: Rename "es" to "elasticsearch"

## 1. Backend

- [x] 1.1 Register inbound route `POST /api/v1/inbound/elasticsearch` with `SourceType: "elasticsearch"` (no /es alias).
- [x] 1.2 Update comments in `internal/inbound/generic.go` and `internal/models/models.go` to say "elasticsearch" where they reference "es".
- [x] 1.3 If migrating existing data: add a one-time step or script to update datasources and/or alerts with type/source_type `"es"` to `"elasticsearch"` (optional; can defer).

## 2. Frontend

- [x] 2.1 In Datasources page type Select, replace option value `"es"` with `"elasticsearch"` (label can be "Elasticsearch").

## 3. Docs and validation

- [x] 3.1 Update `docs/requirements-alert-platform.md` (and any OpenSpec datasources spec) to use "elasticsearch" and document `/inbound/elasticsearch`.
- [x] 3.2 Manually verify: create a datasource of type elasticsearch, list it, and (if applicable) send a test webhook to `/inbound/elasticsearch`.
