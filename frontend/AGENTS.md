# Frontend Agent Notes

## Scope

This folder contains the Vite, React, and TypeScript UI for the local transaction simulation API. Keep the app as the usable first screen; avoid landing-page or marketing-style additions.

## Structure

- `src/app/` owns the entrypoint, app shell, and form state/default builders.
- `src/api/` owns HTTP calls, Zod schemas, and inferred API types.
- `src/features/` owns feature surfaces: `request/`, `output/`, `trace/`, `fund-flow/`, and `balances/`.
- `src/components/` is for small shared UI primitives used by more than one feature.
- `src/lib/` is for pure helpers such as formatting, labels, explorer links, and persistence.
- `src/styles/` contains feature-focused CSS files imported through `src/styles/index.css`.

## Commands

- `yarn lint`
- `yarn build`
- `yarn test:browser`
- `docker compose config`

## Conventions

- Keep feature code in its feature folder and shared code in `components/` or `lib/`; avoid large one-file changes.
- Keep heavy derivation logic out of React components when it can live in a nearby helper file, as with fund-flow grouping and graph layout.
- Keep CSS split by feature, and add new style files through `src/styles/index.css` so load order stays explicit.
- Preserve local input persistence unless a request explicitly changes form behavior.
- Trace output should remain readable: expandable tree, left-aligned rows, wrapped long lines, and collapsible bytes values.
- Fund-flow and balance views should prefer labels over raw addresses, with explorer links and hover details where applicable.
- When backend API response fields change, update `src/api/schemas.ts`, inferred types, and affected feature views together.
- When changing layout behavior, run or update the Playwright browser test in `tests/`.
