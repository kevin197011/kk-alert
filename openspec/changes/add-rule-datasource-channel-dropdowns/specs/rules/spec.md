# Rules – spec delta (datasource and channel selection UI)

## MODIFIED Requirements

### Requirement: Rule match conditions and routing (UI for datasource and channel selection)

Rules SHALL support configurable match conditions (e.g. datasource, labels, severity) and routing to one or more channels. The rule **editing UI** SHALL provide **dropdown selection** for datasource IDs and channel IDs instead of free-text JSON. The options SHALL be loaded from the configured datasources and channels (e.g. from the datasource and channel list APIs). The user SHALL select by name (and optionally type); the form SHALL submit the selected IDs in the same format the backend expects (JSON array of IDs).

#### Scenario: Select datasources from dropdown

- **GIVEN** the user is creating or editing a rule  
- **WHEN** the user opens the Datasource IDs control  
- **THEN** the control shows a multi-select dropdown populated with the configured datasources (e.g. by name); the user selects one or more; on save the rule receives the corresponding IDs as the existing API expects (e.g. JSON array)

#### Scenario: Select channels from dropdown

- **GIVEN** the user is creating or editing a rule  
- **WHEN** the user opens the Channel IDs control  
- **THEN** the control shows a multi-select dropdown populated with the configured channels (e.g. by name); the user selects one or more; on save the rule receives the corresponding IDs as the existing API expects (e.g. JSON array)

#### Scenario: Edit rule shows current selection

- **WHEN** the user opens the rule form for an existing rule that has `datasource_ids` and `channel_ids` set  
- **THEN** the datasource and channel dropdowns show the currently selected items (by ID) so the user can see and change the selection
