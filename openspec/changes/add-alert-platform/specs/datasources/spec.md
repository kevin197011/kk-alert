## ADDED Requirements

### Requirement: Datasource CRUD

The system SHALL support create, read, update, and delete of alert datasources. Each datasource SHALL have at least: name, type (Prometheus, VictoriaMetrics, Elasticsearch, Doris), endpoint, authentication configuration, and enabled/disabled state.

#### Scenario: Create and list datasources

- **WHEN** a user creates a datasource with name, type, and endpoint and then lists datasources
- **THEN** the new datasource appears in the list with the given attributes

#### Scenario: Disable datasource

- **WHEN** a user disables a datasource
- **THEN** the platform no longer routes alerts from that source to channels (or stops accepting from it, per configuration)

### Requirement: Inbound webhook reception

The system SHALL accept alert payloads via webhook endpoints for Prometheus, VictoriaMetrics, Elasticsearch, and Doris, and SHALL normalize each incoming alert into a unified internal model.

#### Scenario: Receive and store alert from Prometheus

- **WHEN** the platform receives a valid webhook payload from a configured Prometheus/Alertmanager
- **THEN** the alert is normalized, stored with a unique alert_id, and made available for rules and history

### Requirement: Datasource connection or ingest test

The system SHALL support a “test connection” or “send test alert” action per datasource so that the operator can verify the platform can receive alerts from that source.

#### Scenario: Test ingest

- **WHEN** the user triggers a test for a configured datasource
- **THEN** the system either confirms connectivity or that a test alert was received and stored (behavior is implementation-defined)
