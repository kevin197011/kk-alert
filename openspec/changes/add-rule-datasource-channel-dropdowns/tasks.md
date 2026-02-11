# Implementation Tasks: Datasource and Channel dropdowns on Rules form

## 1. Frontend

- [x] 1.1 On the Rules page, fetch datasources from `GET /api/v1/datasources` and channels from `GET /api/v1/channels` when the rule modal is opened (or on page load). Store lists in state (or use in form options).
- [x] 1.2 Replace the "Datasource IDs" text input with a multi-select (e.g. Ant Design `Select mode="multiple"`). Options: each datasource as `{ value: id, label: name (and optionally type) }`. Form value: array of IDs; on submit convert to JSON array string `datasource_ids` (e.g. `JSON.stringify(selectedIds)`). Allow empty selection to mean “all datasources” if that is the current semantics.
- [x] 1.3 Replace the "Channel IDs" text input with a multi-select. Options: each channel as `{ value: id, label: name (and optionally type) }`. Form value: array of IDs; on submit convert to JSON array string `channel_ids`. Validate that at least one channel is selected if the product requires it, or allow empty per current behaviour.
- [x] 1.4 When opening the rule form for **edit**, parse the rule’s `datasource_ids` and `channel_ids` (JSON strings) into arrays and set them as the initial selected values for the two dropdowns.
- [x] 1.5 When opening for **add**, default both to empty (or sensible default) so the payload matches existing API expectations.

## 2. Validation

- [x] 2.1 Create a new rule: select one or more datasources and channels from the dropdowns; save. Confirm the saved rule has the correct `datasource_ids` and `channel_ids` (e.g. by editing again or via API). Create an alert and confirm it is matched and sent according to the rule.
- [x] 2.2 Edit an existing rule: change datasource/channel selection, save. Confirm persistence and that rule behaviour reflects the new selection.
