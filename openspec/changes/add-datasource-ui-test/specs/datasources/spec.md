# Datasources – spec delta (UI test action)

## MODIFIED Requirements

### Requirement: Datasource connection or ingest test (UI exposure)

The system SHALL support a "test connection" or "send test alert" action per datasource so that the operator can verify the platform can receive alerts from that source. The datasources list UI SHALL expose a per-row "Test" action (e.g. button) that calls the test API (`POST /api/v1/datasources/:id/test`) and SHALL display the API response (success or error) to the user.

#### Scenario: Test from datasources list

- **GIVEN** the user is on the Datasources page and at least one datasource exists  
- **WHEN** the user clicks the "Test" action for a datasource row  
- **THEN** the client calls the test API for that datasource id and shows the user a success or error message according to the response

#### Scenario: Test failure shown to user

- **WHEN** the user triggers a test and the API returns an error (e.g. 404 or 500)  
- **THEN** the UI displays the error (e.g. "not found" or the server error message) so the user knows the test failed
