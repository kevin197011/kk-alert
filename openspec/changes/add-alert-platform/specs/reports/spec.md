## ADDED Requirements

### Requirement: Report aggregation dimensions

The system SHALL support aggregation of alert history for reporting by at least: time (e.g. hour, day, week, month), datasource, severity, and service/tags. Aggregated counts or metrics SHALL be available via API for charting.

#### Scenario: Aggregate by time and severity

- **WHEN** a user requests a report aggregated by day and severity for a time range
- **THEN** the system returns counts (or equivalent) per day and per severity for that range

### Requirement: Export alerts or report data

The system SHALL support export of current alert list or report data to CSV or Excel. Export SHALL be subject to the same authentication and authorization as the rest of the application.

#### Scenario: Export list to CSV

- **WHEN** an authenticated user exports the current alert list (or report result) as CSV
- **THEN** the system returns a CSV file whose content matches the current list or report data

### Requirement: Reports and export require authentication

The system SHALL allow access to report views and export APIs only for authenticated users; unauthenticated requests SHALL be rejected (e.g. 401).

#### Scenario: Unauthenticated report access

- **WHEN** an unauthenticated client calls the reports or export API
- **THEN** the system responds with 401 or equivalent and does not return data
