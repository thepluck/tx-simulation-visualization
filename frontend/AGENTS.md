# Frontend Agent Notes

## Scope

This folder contains the Vite, React, and TypeScript UI for the local transaction simulation API. Keep the app as the usable first screen; avoid landing-page or marketing-style additions.

## Commands

- `yarn lint`
- `yarn build`
- `yarn test:browser`

## Conventions

- Keep UI split across components in `src/components/`; avoid large one-file changes.
- Preserve local input persistence unless a request explicitly changes form behavior.
- Trace output should remain readable: expandable tree, left-aligned rows, wrapped long lines, and collapsible bytes values.
- Fund-flow and balance views should prefer labels over raw addresses, with explorer links and hover details where applicable.
- When changing layout behavior, run or update the Playwright browser test in `tests/`.
