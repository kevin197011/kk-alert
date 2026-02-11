# Design: Left sidebar and design system

Design system source of truth: **`design-system/kk-alert/MASTER.md`** (colors, typography, effects, anti-patterns).

## Layout

- **Current**: Ant Design `Layout` with `Header` (top) and horizontal `Menu`; `Content` below.
- **Target**: Ant Design `Layout` with `Sider` (left, fixed width or collapsible) and `Content` (right). Optional top bar for brand + logout only.
- **Navigation**: Same six items (Alert History, Rules, Datasources, Channels, Templates, Reports) in vertical `Menu` inside `Sider`. Selected state by route.
- **Space**: Content area has `padding` so it does not sit under the sidebar; reserve space for Sider (e.g. `margin-left` or flex so Content fills remaining width).

## Design system (from ui-ux-pro-max)

- **Style**: Data-dense dashboard; minimal, space-efficient, maximum data visibility.
- **Colors**: Primary #171717, Secondary #404040, CTA/accent #D4AF37, Background #FFFFFF, Text #171717. Use consistently for nav, buttons, cards, tables.
- **Typography**: Fira Sans (body), Fira Code or Fira Sans (headings). Load via Google Fonts; set in global CSS or Ant Design theme.
- **Effects**: Hover tooltips, row/card highlight on hover, smooth filter/transition (150–300ms). No ornate design.
- **Avoid**: Emoji as icons; layout shift when content loads; invisible focus states.

## Tech

- Stack: React + Ant Design + Vite. Use Ant Design’s `Layout.Sider`, `Menu` with `mode="inline"`, and theme token overrides if needed.
- No new dependencies required; optional: Ant Design 5.x theme config for palette.
