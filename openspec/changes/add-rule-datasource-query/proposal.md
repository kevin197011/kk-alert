# Change: Rule query expression per datasource type (PromQL / ES SQL / SQL)

## Why

Rules today match incoming alerts by datasource IDs, labels, and severity. Operators need to express **datasource-specific query logic** where applicable: Prometheus uses **PromQL**, Elasticsearch uses **ES SQL** (or query DSL), and Doris uses **SQL**. The rule form should adapt so that when a rule is scoped to (or primarily for) a given datasource type, the UI shows the appropriate query field and language label (PromQL, ES SQL, or SQL).

## What Changes

- **Rule model**: Add optional fields to store a query expression and its language:
  - `query_language`: one of `promql` | `elasticsearch_sql` | `sql` (or empty when not used).
  - `query_expression`: text field for the query (PromQL, ES SQL, or Doris SQL).
- **Rules UI**: When creating or editing a rule:
  - If the rule is tied to one or more datasources, the UI MAY infer the query language from the selected datasource(s) type (prometheus → PromQL, elasticsearch → ES SQL, doris → SQL). If multiple types are selected, the user selects the query language explicitly.
  - Show a single "Query" area with a **dynamic label and placeholder** based on language: "PromQL", "ES SQL", or "Doris SQL", with a short placeholder example for each.
- **Backend**: Accept and return `query_language` and `query_expression` in rule CRUD; no execution of these queries in this change (storage and UI only). Execution (e.g. periodic evaluation) can be a later change.
- **Docs**: Update PRD/spec to describe that rules MAY include a datasource-type-specific query expression (PromQL / ES SQL / SQL) for reference or future evaluation.

## Impact

- **Affected**: Backend (Rule model, handlers, migration), frontend (Rules form: query field with adaptive label/placeholder), OpenSpec rules spec.
- **Out of scope for this change**: Actually running PromQL/ES SQL/SQL against datasources (pull-based alerting). This change only adds storage and UI for writing the expression.
