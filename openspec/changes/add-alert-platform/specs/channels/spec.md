## ADDED Requirements

### Requirement: Channel CRUD for Telegram and Lark

The system SHALL support create, read, update, and delete of notification channels. Supported channel types SHALL include Telegram and Lark (Feishu). Each channel SHALL have the configuration required by that type (e.g. bot token and chat ID for Telegram; webhook or app credentials for Lark) and an enabled/disabled state.

#### Scenario: Create Telegram channel and send test

- **WHEN** a user creates a Telegram channel with valid token and chat ID and triggers “send test message”
- **THEN** a test message is delivered to the configured Telegram chat

#### Scenario: Create Lark channel and send test

- **WHEN** a user creates a Lark channel with valid configuration and triggers “send test message”
- **THEN** a test message is delivered to the configured Lark target

### Requirement: Secure storage of channel secrets

The system SHALL store channel tokens and secrets in encrypted form and SHALL NOT return plain-text secrets to the frontend (e.g. when editing, support “keep current” or “replace with new value”).

#### Scenario: Secrets not exposed in API response

- **WHEN** a client requests channel details
- **THEN** sensitive fields (tokens, secrets) are omitted or masked in the response

### Requirement: Rules reference channels by ID

The system SHALL allow alert rules to reference channel(s) by ID so that matched alerts can be sent to the selected channel(s).

#### Scenario: Rule routes to channel

- **WHEN** an alert matches a rule that references channel ID C
- **THEN** the platform sends the notification via channel C (subject to other rule options such as rate limit and suppression)
