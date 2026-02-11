## ADDED Requirements

### Requirement: User login with credentials

The system SHALL allow a user to authenticate with username and password and SHALL establish a session (e.g. JWT or server session).

#### Scenario: Login success

- **WHEN** the user submits valid username and password
- **THEN** the system returns a token or session identifier and the user is considered logged in

#### Scenario: Login failure

- **WHEN** the user submits invalid credentials
- **THEN** the system returns an error and does not issue a token or session

### Requirement: Authenticated API access

The system SHALL require a valid token or session for all APIs except login and health checks, and SHALL return 401 when the token is missing or invalid.

#### Scenario: Request without token

- **WHEN** a client calls a protected API without a valid token
- **THEN** the system responds with 401 Unauthorized

#### Scenario: Request with valid token

- **WHEN** a client calls a protected API with a valid token
- **THEN** the system processes the request and returns the appropriate response

### Requirement: Logout

The system SHALL support logout so that the current token or session is invalidated and subsequent requests with that token are rejected.

#### Scenario: Logout invalidates session

- **WHEN** the user logs out
- **THEN** the token or session is invalidated and the next request using it receives 401

### Requirement: Redirect unauthenticated UI access to login

The system SHALL redirect unauthenticated access to configuration, alert history, or reports pages to the login page.

#### Scenario: Unauthenticated access to protected page

- **WHEN** an unauthenticated user navigates to a protected page (e.g. datasources, rules, history, reports)
- **THEN** the user is redirected to the login page
