# Frontend layout – spec delta

## ADDED Requirements

### Requirement: Left sidebar navigation

The application SHALL use a fixed left sidebar for primary navigation. The sidebar SHALL list the same entries as the current top menu: Alert History, Rules, Datasources, Channels, Templates, Reports. The main content area SHALL be rendered to the right of the sidebar and SHALL NOT be obscured by it (content has reserved space).

#### Scenario: Navigate via sidebar

- **GIVEN** the user is logged in and sees the main app layout  
- **WHEN** the user clicks a sidebar item (e.g. Rules)  
- **THEN** the route changes to the corresponding page and the main content area shows that page; the sidebar item is visually selected

#### Scenario: Content not under sidebar

- **GIVEN** the layout with left sidebar is visible  
- **WHEN** any page content is rendered  
- **THEN** the content area has sufficient padding/margin so that no part of the page body is hidden under the sidebar

### Requirement: Consistent design system

The application SHALL apply a single design system for colors, typography, and spacing across the layout and all config/history/reports pages. Colors SHALL be used consistently (e.g. primary, secondary, accent, background, text). Typography SHALL use a defined font set (e.g. Fira Sans for body, Fira Code or Fira Sans for headings). Interactive elements SHALL have visible hover and focus states and smooth transitions (e.g. 150–300 ms).

#### Scenario: Consistent palette

- **GIVEN** the user views the sidebar and any content page  
- **WHEN** comparing navigation, buttons, and tables  
- **THEN** the same color roles (primary, secondary, accent, background, text) are used so the interface looks 色调一致 (consistent in tone)

#### Scenario: Accessible interaction

- **GIVEN** any clickable element in the layout or pages  
- **WHEN** the user hovers or focuses (keyboard)  
- **THEN** the element shows clear visual feedback (e.g. color or border change) and uses cursor-pointer where appropriate; focus is visible for keyboard navigation
