# FRONTEND AGENT KNOWLEDGE BASE (React/TS)

## OVERVIEW
React 18 + TypeScript SPA powered by Vite, providing the project management dashboard.

## STRUCTURE
- `src/App.tsx`: Central router and layout management using `react-router-dom`.
- `src/services/api.ts`: Axios-based API client with interceptors for auth.
- `src/types/index.ts`: Single source of truth for all shared TypeScript interfaces.
- `src/components/`, `src/hooks/`, `src/contexts/`: Placeholder directories (currently empty).
- `src/styles/index.css`: Tailwind CSS entry point.
- `src/main.tsx`: Application entry point.

## CONVENTIONS

### Import Ordering
1. React and standard hooks (e.g., `useState`, `useEffect`)
2. Third-party libraries (e.g., `axios`, `zustand`, `@dnd-kit`)
3. Local modules/components using `@/` path alias
4. CSS and type definitions

### TypeScript
- **Strict Mode**: Fully enabled in `tsconfig.json`. No `any` allowed.
- **Types**: Prefer `interface` for object definitions.
- **Location**: All domain models/shared types must reside in `src/types/index.ts`.

### Components & State
- **Functional Only**: Use functional components with hooks (no class components).
- **Hooks**: Use custom hooks in `src/hooks/` for complex logic or API data fetching.
- **State**: Use `zustand` for global state and React `Context` for UI-specific state.

### Linting & Styling
- **ESLint**: Strict zero-warning policy (`--max-warnings 0`) enforced in CI.
- **Styling**: Tailwind CSS utility classes only. 
- **Responsive**: Use `sm:`, `md:`, `lg:` prefixes for mobile-first design.

## COMMANDS
```bash
npm run dev           # Start Vite dev server on :5173
npm run lint          # Run ESLint (strict mode)
npm run format        # Run Prettier formatting
npm run build         # Production build (runs tsc + vite build)
npm test              # Run Vitest suite
npm test -- <path>    # Run specific test file
npm test -- --watch   # Run tests in watch mode
```

## GOTCHAS
- **Path Alias**: `@/` maps to `src/`. Ensure synchronization with `tsconfig.json`.
- **Prettier**: No `.prettierrc` exists; uses defaults. Watch for ESLint conflicts.
- **Directories**: `components/`, `hooks/`, and `contexts/` exist but are currently unpopulated.
- **Environment**: Ensure `.env` is configured for API base URL (default: `http://localhost:8080`).
