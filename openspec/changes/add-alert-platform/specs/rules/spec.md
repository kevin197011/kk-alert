## ADDED Requirements

### Requirement: Rule CRUD and enable/disable

The system SHALL support create, read, update, and delete of alert rules. Each rule SHALL have an enabled/disabled (active/inactive) state. When a rule is disabled, it SHALL NOT match alerts or send to channels; when enabled, it SHALL participate in matching and sending. The list and detail views SHALL show the current state.

#### Scenario: Disabled rule does not send

- **WHEN** a rule is disabled and an alert matches that rule’s conditions
- **THEN** the platform does not send the alert to the rule’s channels

#### Scenario: Enable rule and send

- **WHEN** a rule is re-enabled and a new alert matches it
- **THEN** the platform sends the alert according to the rule (subject to other constraints)

### Requirement: Rule match conditions and routing

The system SHALL support configurable match conditions per rule (e.g. datasource, labels, severity, keywords) and SHALL route matched alerts to one or more channels using a designated template. Rules SHALL be evaluated in a defined order (e.g. priority); the first matching rule or all matching rules may apply (implementation-defined).

#### Scenario: Match and route to channel

- **WHEN** an alert matches a rule’s conditions and the rule references channel C and template T
- **THEN** the platform sends the alert to channel C using template T (subject to duration, windows, rate limit, suppression)

### Requirement: Check frequency and duration threshold

The system SHALL allow each rule to define an evaluation/check frequency (e.g. 30s, 1m, 5m) and an optional duration threshold. The system SHALL send only when the alert has been continuously matching for at least the duration (or immediately if duration is zero or unset).

#### Scenario: Duration threshold delays send

- **WHEN** a rule has duration 5m and an alert first matches at T0
- **THEN** the platform sends no notification until the alert has been matching for 5 minutes (or implementation-defined equivalent)

### Requirement: Exclude time windows

The system SHALL allow each rule to define one or more “do not send” time windows (e.g. daily 22:00–08:00, or weekly maintenance). During these windows, matched alerts SHALL NOT be sent to channels but SHALL still be stored and visible in history.

#### Scenario: No send inside exclude window

- **WHEN** current time is inside a rule’s exclude window and an alert matches the rule
- **THEN** the platform does not send to channels; the alert is still stored and queryable in history

### Requirement: Alert recovery notification

The system SHALL allow each rule to optionally enable a “recovery” notification. When an alert transitions from firing to resolved, if the rule has recovery enabled, the system SHALL send a recovery message to the same channel(s), distinguishable from the firing message (e.g. title or body indicating “recovered”).

#### Scenario: Recovery notification sent

- **WHEN** an alert goes from firing to resolved and its matching rule has recovery enabled
- **THEN** the platform sends a recovery notification to the rule’s channel(s)

### Requirement: Send rate limit per rule

The system SHALL allow each rule to configure send rate limits (e.g. minimum interval per alert_id, or max sends per hour per rule). Alerts that exceed the rate limit SHALL NOT be sent to channels but SHALL still be stored and visible in history.

#### Scenario: Rate limit suppresses repeat send

- **WHEN** a rule has “same alert at most once per 5 minutes” and the same alert_id triggers again within 5 minutes
- **THEN** the platform does not send a second notification; the alert is still stored

### Requirement: Same-type aggregation by hostname, IP, or port

The system SHALL support per-rule aggregation of “same type” alerts (same rule and same labels except the aggregation dimension) by dimension: hostname, host IP, or port. Within a configurable time window, multiple such alerts SHALL be merged into one notification that includes the list (or count) of hosts/IPs/ports. Each original alert SHALL still be stored with its own alert_id.

#### Scenario: Aggregated send by hostname

- **WHEN** a rule has hostname aggregation enabled and five alerts of the same type for five different hosts arrive within the aggregation window
- **THEN** the platform sends one notification that includes the five hosts (or a summary such as “5 hosts”) and all five alerts remain in history with distinct alert_ids

### Requirement: Tag-based suppression

The system SHALL support tag-based suppression: when an alert matches a “source” condition, a time window starts during which alerts matching a “suppressed” condition are not sent to channels (they are still stored and visible in history). Suppression rules SHALL be configurable (source condition, suppressed condition, duration).

#### Scenario: Suppressed alert not sent

- **WHEN** a source-matching alert has started a suppression window and a new alert matches the suppressed condition within that window
- **THEN** the platform does not send the suppressed alert to channels; it is still stored

### Requirement: Jira ticket on N occurrences

The system SHALL allow each rule to optionally create a Jira ticket when an alert has “fired” or been observed N times (e.g. same alert_id or same fingerprint). The system SHALL support Jira configuration (project, issue type, summary/description template, credentials stored securely). The same alert SHALL create at most one ticket (or per implementation: only when no linked ticket exists). Creation failure SHALL be logged and SHALL NOT block alert storage or sending.

#### Scenario: Jira ticket created after N occurrences

- **WHEN** a rule has “create Jira after 3 occurrences” and the same alert reaches 3 occurrences
- **THEN** the platform creates a Jira ticket (if not already created for that alert) and may associate the ticket key with the alert/history

### Requirement: Rule JSON import and export

The system SHALL support exporting one or more rules as JSON (including full configuration: match, route, frequency, duration, windows, recovery, rate limit, aggregation, suppression, Jira). The system SHALL support importing rules from JSON with strategies such as “add only” or “overwrite by name”; invalid references (e.g. missing channel or template ID) SHALL be reported and SHALL NOT be applied.

#### Scenario: Export and re-import rules

- **WHEN** a user exports several rules to JSON and then imports that JSON (e.g. into another environment or after reset)
- **THEN** the imported rules behave the same as before export, provided referenced channels and templates exist

### Requirement: Batch operations on rules

The system SHALL support batch enable, batch disable, and batch delete on selected rules. The result SHALL report success and failure counts and SHALL indicate reasons for failures (e.g. rule in use).

#### Scenario: Batch disable

- **WHEN** a user selects multiple rules and performs batch disable
- **THEN** only the selected rules are disabled and the UI shows the updated state
