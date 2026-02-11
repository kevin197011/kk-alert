# Implementation Tasks: Left sidebar layout and design system

## 1. Layout

- [x] 1.1 Replace top horizontal menu with a fixed left sidebar (Ant Design Layout Sider).
- [x] 1.2 Sidebar contains navigation links: Alert History, Rules, Datasources, Channels, Templates, Reports (same entries as today, vertical list).
- [x] 1.3 Main content area has correct padding and does not sit under the sidebar; reserve space for sidebar (no overlap).
- [x] 1.4 Optional: collapsible sidebar (icon-only when collapsed) for more content space.

## 2. Design system

- [x] 2.1 Introduce a single color palette (e.g. primary #171717, secondary #404040, accent #D4AF37, background #FFFFFF, text #171717) and use it across Layout and pages.
- [x] 2.2 Apply consistent typography (e.g. Fira Sans for body, Fira Code or Fira Sans for headings) via CSS/theme.
- [x] 2.3 Use consistent spacing (e.g. content padding, gap between sidebar and content) and avoid layout shift when content loads.
- [x] 2.4 Ensure hover/focus states on nav and buttons: cursor-pointer, smooth transitions (150–300ms), visible focus for keyboard.

## 3. Validation

- [x] 3.1 All existing routes and pages work with the new layout (no broken links or hidden content).
- [x] 3.2 Responsive: sidebar behavior acceptable on 1024px and 1440px; optional collapse or drawer on smaller widths.
- [x] 3.3 No emojis as icons; use SVG (e.g. Ant Design icons or Lucide) where needed.
