<!-- GSD:project-start source:PROJECT.md -->
## Project

**Mitti Exporter Monorepo**

A monorepo consolidation of the Mitti Exporter CLI (rebranded from SafetyCulture Exporter, see FG-7681) and its desktop UI wrapper. The CLI (Go) and UI (Wails + React/Tailwind/shadcn) live side-by-side using Go workspaces, with the UI referencing the CLI as a local module instead of published build artifacts. A unified CI pipeline publishes both CLI binaries and desktop app bundles as GitHub Release assets.

**Core Value:** One repository, one release process — the UI always builds against the latest CLI code without waiting for artifact publishing.

### Constraints

- **Go workspaces**: Must use `go.work` for local module references — no vendoring hacks
- **Module path**: CLI module path is `github.com/SafetyCulture/mitti-exporter` (changed from `github.com/SafetyCulture/safetyculture-exporter` as a deliberate breaking change during the Mitti rebrand — external Go consumers must update their import path; this is a major version bump). Once set, do not change it again without another explicit decision.
- **Frontend stack**: React + Tailwind CSS + shadcn/ui with pnpm (not npm) and Vite
- **Wails v2**: UI must remain a Wails v2 desktop application
- **Release**: Single GitHub Release with both CLI and UI artifacts
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## Recommended Stack
### Core Technologies
| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24.x (toolchain 1.24.10) | Backend language + CLI | Current stable; CLI already targets `go 1.23` with `toolchain go1.24.1` — no upgrade needed. Go 1.24 added generic type aliases and `tool` directives in `go.mod`. |
| Go Workspaces (`go.work`) | Go 1.18+ (supported in 1.24) | Local multi-module dev | The canonical way to have `ui/` reference `cli/` without publishing CLI artifacts. Eliminates `replace` directives in `go.mod`. |
| Wails v2 | v2.12.0 | Desktop app framework | Latest stable v2 release (Mar 2026). v3 is in alpha — not suitable for production. v2.10.0 has known build action issues; v2.12.0 fixes clipboard, WebView stability, and Linux panics. |
| React | 19.2.x | Frontend UI library | Latest stable (19.2.0, Oct 2025). Required by shadcn/ui components updated for React 19. Aligns with internal `web-platform` reference which uses `react: ^19.2.4`. |
| TypeScript | 5.6.x | Type safety for frontend | Compatible with `@vitejs/plugin-react ^4.3.3`. The internal web-platform uses TypeScript 6 but Wails' react-ts template ecosystem is validated on TS 5.x — use 5.6.x to avoid breaking Wails bindings codegen. |
| Vite | 5.4.x | Frontend build tool + dev server | **Critical version constraint**: Vite 5.0.0+ broke a Wails v2 dynamic asset feature; Vite 7.1.8 broke the dev server connection entirely (issue #4620, though fixed in 7.1.9). Safest choice for Wails v2 in 2026 is **Vite 5.4.x** — fully supported, HMR works with the documented `hmr.protocol: "ws"` fix. |
| Tailwind CSS | v4.2.x | Utility CSS framework | Stable since Jan 2025. shadcn/ui now supports Tailwind v4 natively. New `@import "tailwindcss"` replaces `@tailwind` directives. Uses `@tailwindcss/vite` plugin (not PostCSS). Aligns with `web-platform` (`tailwindcss: ^4.2.2`). |
| shadcn/ui | latest CLI (`shadcn@latest`) | Component library | Not a versioned package — components are copied into the project via CLI. All components updated for Tailwind v4 + React 19 as of 2025. Provides unstyled, accessible primitives via Radix UI. |
| pnpm | 10.x (10.23+) | Node package manager | Required by project constraints. Faster, stricter than npm. Aligns with `web-platform`. Node.js >= 18.12 required. Use `"frontend:install": "pnpm install"` in `wails.json`. |
### Supporting Libraries
| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| `@vitejs/plugin-react` | ^4.3.3 | Vite React plugin | The standard Babel-based React plugin for Vite. Works with Vite 5.x. Use `plugin-react` (Babel) not `plugin-react-swc` — Wails dev server reload is simpler with the default. |
| `@tailwindcss/vite` | ^4.x | Tailwind v4 Vite integration | Tailwind v4 dropped PostCSS in favour of a native Vite plugin. Add to `plugins: [react(), tailwindcss()]` in `vite.config.ts`. |
| `@types/react` | ^19.x | React TypeScript types | Match React 19.x peer dep. |
| `@types/react-dom` | ^19.x | React DOM TypeScript types | Match React 19.x peer dep. |
| `@types/node` | ^20.x | Node types for Vite config | Required for `path.resolve(__dirname, ...)` in `vite.config.ts`. |
| `tailwind-merge` | ^3.x | Merge Tailwind classes | Used by shadcn/ui internals. Aligns with `web-platform` (`tailwind-merge: ^3.5.0`). |
| `clsx` | ^2.x | Class name utility | Used by shadcn/ui `cn()` helper. |
| `lucide-react` | ^0.x (latest) | Icon set | shadcn/ui default icon library. |
| `class-variance-authority` | ^0.7.x | shadcn/ui variant API | Required by shadcn/ui component generation. |
### Development Tools
| Tool | Version | Purpose | Why |
|------|---------|---------|-----|
| `wails` CLI | v2.12.0 | Project scaffold, dev server, build | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| `wails generate module` | (bundled with wails CLI) | Regenerate TS bindings without full dev cycle | Run when Go struct signatures change. |
| `The-Egg-Corp/wails-build-action` | @v2 | GitHub Actions Wails builder | More actively maintained fork of `dAppServer/wails-build-action`. Supports pnpm via pre-steps. |
| `pnpm/action-setup` | @v4 | Install pnpm in CI | Official pnpm GitHub Action. Must run before `actions/setup-node` to enable pnpm caching. |
| `actions/setup-node` | @v4 | Node.js in CI | Use with `cache: 'pnpm'` after pnpm/action-setup. |
| `actions/setup-go` | @v5 | Go in CI | Required for the wails-build-action Go version override. |
## Installation
### 1. Repo Structure
### 2. Go Workspace (`go.work`)
### 3. `wails.json` (in `ui/`)
- `"frontend:dev:serverUrl": "auto"` — Wails reads the URL from Vite's stdout output automatically.
- `"wailsjsdir": "./frontend/src/wailsjs"` — places generated bindings inside `src/` so TypeScript path resolution works naturally.
- `"frontend:install": "pnpm install"` — Wails runs this when `node_modules` is missing.
### 4. `vite.config.ts` (in `ui/frontend/`)
### 5. `ui/frontend/src/index.css`
### 6. `ui/frontend/tsconfig.json`
### 7. `ui/frontend/tsconfig.app.json`
### 8. Initialize shadcn/ui
### 9. Wails JS Bindings Pattern
## GitHub Actions CI
### Strategy
- **CLI build/test**: Run with `GOWORK=off` so each module builds against its own `go.mod` (simulates external consumer behaviour).
- **UI build**: Run with the workspace active so `ui/` resolves `cli/` locally.
- **Cross-platform UI builds**: Matrix across `macos-latest` (darwin/universal) and `windows-latest` (windows/amd64). macOS cannot be cross-compiled — each platform must build natively.
### Workflow: `.github/workflows/release.yml`
- Using `GOWORK=off` for CLI builds ensures the CLI's `go.mod` is authoritative (module isolation for external consumers).
- The UI build runs without disabling the workspace — Wails needs `go.work` in place to find the local `cli/` module.
- `pnpm/action-setup@v4` before `actions/setup-node@v4` is the required order for pnpm caching to work in GitHub Actions.
- `darwin/universal` produces a fat binary (amd64 + arm64) for both Intel and Apple Silicon Macs. This requires building on `macos-latest`.
- The `dAppServer/wails-build-action` composite action is an alternative to this manual workflow but does not natively support pnpm. The manual workflow above gives full control.
## Module Path Preservation Strategy
## Alternatives Considered
| Alternative | Verdict | Reason Rejected |
|-------------|---------|-----------------|
| **Wails v3 (alpha)** | Rejected | API unstable, documentation incomplete, not production-ready. v2.12.0 is the safe choice through 2026. |
| **`replace` directive in `ui/go.mod`** | Rejected | Go workspaces supersede this; `replace` directives are path-fragile and require removal before publishing. |
| **Single `go.mod` for both CLI and UI** | Rejected | Would change the CLI module path; breaks external consumers; entangles CGO (Wails) deps with pure-Go CLI. |
| **Vite 6.x / 7.x** | Rejected | Active compatibility issues with Wails v2 dev server (issue #4620). Vite 5.4.x is the proven-safe version for Wails v2. |
| **npm instead of pnpm** | Rejected | Project constraint; also, pnpm's strict hoist behaviour catches phantom dependencies. |
| **Nx or Turborepo** | Rejected | Explicitly out of scope per PROJECT.md. Go workspaces + pnpm workspaces is sufficient for a two-module repo. |
| **Electron instead of Wails** | Rejected | Out of scope; Wails is a project constraint. |
| **Tailwind CSS v3** | Rejected | shadcn/ui is actively moving to v4. v3 support is legacy-only. Tailwind v4 is stable since Jan 2025. |
| **`@vitejs/plugin-react-swc`** | Rejected | Marginally faster builds but adds a native dependency. Standard `plugin-react` (Babel) has broader Wails issue report coverage. |
## What NOT to Use
- **Vite 5.0.0 with dynamic assets feature** — broken in Wails v2. If you need dynamic assets, this rules out Vite 5.0.0 (but 5.4.x static asset serving is fine for normal React SPA use).
- **Vite 7.1.8 specifically** — dev server connection bug (fixed in 7.1.9, but sticking with 5.4.x avoids the risk).
- **`go work` with `GOWORK=on` in CLI-only CI jobs** — the workspace would pull in Wails CGO deps during CLI linting/testing, complicating runners.
- **`dAppServer/wails-build-action` without a pre-step for pnpm** — the action only sets up npm/Node natively; running `pnpm install` as a pre-step is required.
- **Committing `node_modules/`** — pnpm uses a content-addressable store; only `pnpm-lock.yaml` is committed.
- **PostCSS for Tailwind v4** — Tailwind v4 moved away from PostCSS. Use `@tailwindcss/vite` plugin only.
- **`tailwind.config.js`** — Tailwind v4 is configured entirely via CSS `@theme` directives in `index.css`; no JS config file is used.
- **`wailsjsdir` pointing outside `src/`** — TypeScript path aliasing won't resolve it cleanly; keep bindings inside `src/wailsjs/`.
## Version Compatibility Matrix
| Component | Minimum | Recommended | Notes |
|-----------|---------|-------------|-------|
| Go | 1.18 | 1.24.x | 1.18 introduced `go.work`; 1.24 is current stable |
| Wails v2 | 2.9.0 | 2.12.0 | 2.10.0 has known build action issues; 2.12.0 latest stable |
| Vite | 4.x | 5.4.x | 5.0.0+ breaks one Wails dynamic asset feature; 7.1.8 breaks dev server. Stay on 5.4.x for v2. |
| `@vitejs/plugin-react` | 4.0.0 | 4.3.x | Validated with Vite 5.x and TS 5.6 |
| React | 18.x | 19.2.x | shadcn/ui components updated for React 19 |
| TypeScript | 5.0 | 5.6.x | Use 5.6 not 6.x — TS 6 ecosystem tooling still catching up |
| Tailwind CSS | 4.0.0 | 4.2.x | Stable since Jan 2025 |
| pnpm | 8.x | 10.23.x | Node >= 18.12 required for v10 |
| Node.js | 18.12 | 20.x LTS | LTS preferred in CI; pnpm v10 minimum |
| shadcn/ui | CLI latest | `shadcn@latest` | Not versioned; init installs current component set |
## Sources
- [Wails v2 Project Config Reference](https://wails.io/docs/reference/project-config/)
- [Wails Application Development Guide](https://wails.io/docs/guides/application-development/)
- [Wails Cross-Platform Build with GitHub Actions](https://wails.io/docs/guides/crossplatform-build/)
- [Wails GitHub Releases](https://github.com/wailsapp/wails/releases)
- [Wails Issue #4620 — Vite 7.1.8 dev server incompatibility](https://github.com/wailsapp/wails/issues/4620)
- [Wails Issue #3845 — Vite 5 HMR fix](https://github.com/wailsapp/wails/issues/3845)
- [dAppServer/wails-build-action README](https://github.com/dAppServer/wails-build-action/blob/main/README.md)
- [shadcn/ui Vite Installation](https://ui.shadcn.com/docs/installation/vite)
- [Tailwind CSS v4.0 Release](https://tailwindcss.com/blog/tailwindcss-v4)
- [Go Workspaces Tutorial](https://go.dev/doc/tutorial/workspaces)
- [How to Use Go Workspaces for Monorepos](https://oneuptime.com/blog/post/2026-02-01-go-workspaces-monorepos/view)
- [Go Workspace Structure — Rost Glukhov](https://www.glukhov.org/post/2025/12/go-workplace-structure/)
- [pnpm/action-setup GitHub Action](https://github.com/marketplace/actions/setup-pnpm)
- [pnpm Continuous Integration](https://pnpm.io/continuous-integration)
- [pnpm 2025 Year in Review](https://pnpm.io/blog/2025/12/29/pnpm-in-2025)
- [React v19 Release](https://react.dev/blog/2024/12/05/react-19)
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
