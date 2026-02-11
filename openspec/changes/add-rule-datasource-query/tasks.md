# Implementation Tasks: Rule query per datasource type

## 1. Backend

- [x] 1.1 Add to Rule model: `query_language` (string, e.g. `promql` | `elasticsearch_sql` | `sql`, empty allowed) and `query_expression` (text). Run migration.
- [x] 1.2 Rule CRUD (create/update/list/get) accept and return `query_language` and `query_expression`. Export/import rules include these fields.

## 2. Frontend

- [x] 2.1 Rules form: add a "Query language" selector (optional) with options: PromQL (Prometheus), ES SQL (Elasticsearch), SQL (Doris), and "None". When "None", hide or disable the query text area.
- [x] 2.2 Add a "Query" text area. Label and placeholder SHALL adapt to the selected query language: "PromQL" with placeholder e.g. `up == 0`; "ES SQL" with placeholder e.g. `SELECT * FROM index WHERE ...`; "SQL" (Doris) with placeholder e.g. `SELECT ...`.
- [x] 2.3 Optional: when user selects or changes "Datasource IDs", if all selected datasources are the same type, auto-set query language (prometheus → PromQL, elasticsearch → ES SQL, doris → SQL). Requires loading datasource list and types.

## 3. Docs and spec

- [x] 3.1 Update rules spec delta: rule MAY have a datasource-type-specific query (PromQL / ES SQL / SQL); UI SHALL show the appropriate label and input for the selected language.
- [x] 3.2 Update `docs/requirements-alert-platform.md` (or relevant section) to mention that rules can include an optional query expression (PromQL for Prometheus, ES SQL for Elasticsearch, SQL for Doris).

## 4. Validation

- [x] 4.1 Create a rule with query_language "promql" and a sample PromQL expression; save and reload; confirm values persist. Repeat for ES SQL and SQL.
