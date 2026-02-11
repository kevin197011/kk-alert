# Datasources – spec delta (rename es to elasticsearch)

## MODIFIED Requirements

### Requirement: Datasource type values

The system SHALL support datasource types including Prometheus, VictoriaMetrics, **Elasticsearch**, and Doris. The stored and displayed type value for Elasticsearch SHALL be **"elasticsearch"** (not "es"). Create, read, update, and list APIs SHALL use "elasticsearch" for this type.

#### Scenario: Create datasource with type Elasticsearch

- **GIVEN** the user is on the Datasources page  
- **WHEN** the user creates a datasource and selects type "Elasticsearch" (value "elasticsearch")  
- **THEN** the datasource is stored with type "elasticsearch" and appears in the list with that type

### Requirement: Inbound webhook for Elasticsearch

The system SHALL accept alert payloads for Elasticsearch via `POST /api/v1/inbound/elasticsearch`. Incoming alerts SHALL be stored with `source_type: "elasticsearch"`.

#### Scenario: Receive alert via Elasticsearch webhook

- **WHEN** the platform receives a valid JSON payload at `POST /api/v1/inbound/elasticsearch` (with an `alerts` array)  
- **THEN** each alert is normalized, stored with `source_type: "elasticsearch"`, and made available for rules and history
