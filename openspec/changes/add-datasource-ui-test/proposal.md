# Change: Add datasource test action in UI

## Why

The backend already exposes `POST /api/v1/datasources/:id/test` (test connection) per the existing datasources spec. The datasources list page currently has only Edit and Delete per row. Adding a "Test" action per datasource lets the operator trigger the test from the UI and see success or failure feedback without leaving the list.

## What Changes

- **Frontend**: On the Datasources page, add a "Test" button (or link) in the Actions column for each datasource row. When clicked, the client SHALL call `POST /api/v1/datasources/:id/test` (with auth). The UI SHALL display the API result (e.g. success message or error) to the user (e.g. via message.success / message.error or similar).
- **Backend**: No change required; existing TestConnection handler and route remain as-is.
- **Spec**: Clarify that the datasources UI SHALL expose this test action per row (delta under datasources capability).

## Impact

- **Affected**: Frontend only (Datasources.tsx). Optional: spec delta in datasources.
- **Existing**: POST /api/v1/datasources/:id/test already returns 200 + { ok, message } or 404.
