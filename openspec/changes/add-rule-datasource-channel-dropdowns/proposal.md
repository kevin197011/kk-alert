# Change: Rule form – Datasource IDs and Channel IDs as dropdowns

## Why

Today the Rules form asks users to type **Datasource IDs** and **Channel IDs** as JSON arrays (e.g. `[1,2]`). This is error-prone and requires knowing IDs. The form SHOULD offer **dropdown (multi-select)** controls populated from the configured datasources and channels so users pick by name and the correct ID array is submitted.

## What Changes

- **Rules page (frontend only)**:
  - **Datasource IDs**: Replace the free-text input with a **multi-select** dropdown. Options SHALL be loaded from `GET /api/v1/datasources` (id + name, and optionally type). Selected values SHALL be submitted as the same JSON array of IDs the backend expects (e.g. `[1,2]` or empty for “all”).
  - **Channel IDs**: Replace the free-text input with a **multi-select** dropdown. Options SHALL be loaded from `GET /api/v1/channels` (id + name, and optionally type). Selected values SHALL be submitted as the JSON array of IDs the backend expects.
- **Backend**: No change. Rule model and API continue to accept and return `datasource_ids` and `channel_ids` as JSON strings (array of IDs).
- **Behaviour**: When opening the rule form (add or edit), load datasources and channels once (or when the modal opens). When editing, pre-select the IDs from the rule’s current `datasource_ids` and `channel_ids`. Allow “no selection” for datasource IDs to mean “all datasources” (empty array) where applicable.

## Impact

- **Affected**: Frontend Rules page only (`Rules.tsx`). Optional: rules spec delta to require dropdown selection from configured datasources/channels.
- **APIs**: Existing `GET /api/v1/datasources` and `GET /api/v1/channels` are used; no new endpoints.
