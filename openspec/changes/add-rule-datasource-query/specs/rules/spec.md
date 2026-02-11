# Rules – spec delta (datasource-type-specific query)

## ADDED Requirements

### Requirement: Optional query expression per datasource type

A rule MAY include an optional query expression whose syntax depends on the datasource type. The system SHALL support storing and editing:

- **Prometheus**: query language **PromQL**; the expression is stored and the UI SHALL show the field labeled as "PromQL" with appropriate placeholder (e.g. PromQL example).
- **Elasticsearch**: query language **ES SQL** (or Elasticsearch SQL); the UI SHALL show the field labeled as "ES SQL" with appropriate placeholder.
- **Doris**: query language **SQL**; the UI SHALL show the field labeled as "SQL" (or "Doris SQL") with appropriate placeholder.

The rule SHALL store at least: `query_language` (one of promql, elasticsearch_sql, sql, or empty) and `query_expression` (text). When `query_language` is empty, the query expression is not used. Execution of the query against the datasource (pull-based alerting) is out of scope for this requirement; this requirement covers storage and UI only.

#### Scenario: Edit rule with PromQL

- **GIVEN** the user is creating or editing a rule and selects query language "PromQL"  
- **WHEN** the user enters a PromQL expression (e.g. `up == 0`) and saves  
- **THEN** the rule stores `query_language: "promql"` and `query_expression` with the entered text; the form shows the field labeled "PromQL"

#### Scenario: Edit rule with ES SQL

- **GIVEN** the user selects query language "ES SQL" (Elasticsearch)  
- **WHEN** the user enters an ES SQL statement and saves  
- **THEN** the rule stores `query_language: "elasticsearch_sql"` and the expression; the form shows the field labeled "ES SQL"

#### Scenario: Edit rule with SQL (Doris)

- **GIVEN** the user selects query language "SQL" (Doris)  
- **WHEN** the user enters a SQL statement and saves  
- **THEN** the rule stores `query_language: "sql"` and the expression; the form shows the field labeled "SQL" or "Doris SQL"
