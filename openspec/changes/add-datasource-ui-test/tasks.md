# Implementation Tasks: Datasource test action in UI

## 1. Frontend

- [x] 1.1 Add a "Test" button in the Actions column of the Datasources table for each row (next to Edit and Delete).
- [x] 1.2 On click, call `POST /api/v1/datasources/{id}/test` with auth headers. On success (2xx), show a success message (e.g. "Test passed" or the API message). On error (4xx/5xx), show the error message to the user.
- [x] 1.3 Optionally disable the button or show loading state while the request is in flight to avoid double-clicks.

## 2. Validation

- [x] 2.1 Manually verify: open Datasources, click Test on a datasource; confirm success toast. Click Test on a non-existent or invalid id; confirm error is shown.
