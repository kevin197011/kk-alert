# Change: Rename datasource type "es" to "elasticsearch"

## Why

The short name "es" for Elasticsearch is ambiguous and inconsistent with the full product name. Renaming to "elasticsearch" in datasources (type value, UI, API, and inbound webhook) improves clarity and aligns with common naming.

## What Changes

- **Datasource type value**: Use `"elasticsearch"` instead of `"es"` when creating/editing datasources and when storing type. UI and API SHALL show and accept `"elasticsearch"`.
- **Inbound webhook**: Expose `POST /api/v1/inbound/elasticsearch` with `SourceType: "elasticsearch"` (no `/es` alias).
- **Backend**: Update handlers, models comments, and inbound registration to use the new type and path.
- **Frontend**: Datasources page type dropdown SHALL list `"elasticsearch"` instead of `"es"`.
- **Docs**: Update `docs/requirements-alert-platform.md` (and any other references) to say "elasticsearch" and document the `/inbound/elasticsearch` path.

## Impact

- **Affected**: Backend (main.go, inbound/generic.go, models comment), frontend (Datasources.tsx), docs.
- **Existing data**: Datasources or alerts with type/source_type `"es"` may be migrated to `"elasticsearch"` in a one-time step, or left as-is with the engine accepting both for matching (implementation choice).
- **Breaking**: Callers must use `POST /api/v1/inbound/elasticsearch` (no `/es` path).
