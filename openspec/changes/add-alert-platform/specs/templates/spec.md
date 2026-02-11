## ADDED Requirements

### Requirement: Template CRUD with tag-based content

The system SHALL support create, read, update, and delete of alert templates. Templates SHALL be primarily tag-based: placeholders SHALL reference alert labels/tags (e.g. by key such as `{{.Labels.instance}}`) and MAY reference common fields (e.g. `{{.AlertID}}`, `{{.Title}}`, `{{.Severity}}`, `{{.FiringAt}}`). Missing or empty tag keys SHALL render as empty string or a defined placeholder without failing the send.

#### Scenario: Render template with tags

- **WHEN** an alert with labels `job=api`, `instance=host1` is rendered with a template containing `{{.Labels.job}}` and `{{.Labels.instance}}`
- **THEN** the output contains `api` and `host1` in the corresponding positions

#### Scenario: Missing tag in template

- **WHEN** the template references `{{.Labels.missing_key}}` and the alert has no such label
- **THEN** the render result contains an empty string or defined placeholder and the notification is still sent

### Requirement: Template preview with sample data

The system SHALL support preview of a template using sample alert data (including sample tags) so that users can verify tag placeholders and layout before saving.

#### Scenario: Preview template

- **WHEN** the user requests a preview for a template with sample data that includes labels
- **THEN** the system returns the rendered content as it would appear for an alert with that data

### Requirement: Rules reference template by ID

The system SHALL allow alert rules to reference a template by ID for rendering alert content when sending to channels.

#### Scenario: Rule uses template

- **WHEN** an alert matches a rule that references template T
- **THEN** the notification body is produced by rendering template T with that alert’s data
