# Change: Left sidebar layout and consistent design system

## Why

The current admin UI uses a top horizontal menu (Header + Menu). Moving navigation to a left sidebar improves scannability, fits more menu items without crowding, and aligns with common dashboard patterns. Applying a single design system (colors, typography, spacing) across the app makes the interface 简洁优雅、色调一致 (simple, elegant, consistent).

## What Changes

- **Layout**: Replace top horizontal navigation with a fixed left sidebar. Main content area sits to the right of the sidebar. Header may remain as a slim bar (brand + user/logout) or be merged into the sidebar.
- **Design system**: Apply a consistent visual system for the whole app:
  - Color palette (primary, secondary, background, text) used consistently.
  - Typography (e.g. one font for headings, one for body).
  - Spacing and content padding.
  - No new backend or API changes.

## Impact

- Affected: frontend only (React + Ant Design).
- Reference: current `frontend/src/components/Layout.tsx` (Header + horizontal Menu); design system recommendations from ui-ux-pro-max (data-dense dashboard, minimal black + accent, Fira Code/Sans).
- No auth, API, or backend changes.
