# Mitti Exporter

[![Maintainability](https://api.codeclimate.com/v1/badges/39eecd9ef3573ecca044/maintainability)](https://codeclimate.com/github/SafetyCulture/mitti-exporter/maintainability) [![Test Coverage](https://api.codeclimate.com/v1/badges/39eecd9ef3573ecca044/test_coverage)](https://codeclimate.com/github/SafetyCulture/mitti-exporter/test_coverage)

Mitti Exporter is available as both a **command-line tool (CLI)** and a **desktop application (UI)**. You can use it to export your data — inspections, templates, schedules, actions, and more — to multiple formats for business intelligence tools or record keeping. Available to all Premium and Enterprise customers.

For instructions on downloading and running Mitti Exporter, as well as interpreting the data output, please check out our [Mitti Exporter documentation](https://developer.mitti.com/docs/mitti-exporter).

* [Mitti Exporter documentation](https://developer.mitti.com/docs/mitti-exporter)
* [Run Mitti Exporter](https://developer.mitti.com/docs/mitti-exporter-run)
  * [Download](https://developer.mitti.com/docs/mitti-exporter-run#download)
  * [Configuration](https://developer.mitti.com/docs/mitti-exporter-run#configure)
  * [Database support](https://developer.mitti.com/docs/mitti-exporter-database-support)
  * [Docker support](https://developer.mitti.com/docs/mitti-exporter-docker-support)
  * [Errors](https://developer.mitti.com/docs/mitti-exporter-errors)
* [Understand the data](https://developer.mitti.com/docs/mitti-exporter-data)
  * [CSV or SQL?](https://developer.mitti.com/docs/mitti-exporter-csv-or-sql)
  * [Inspections](https://developer.mitti.com/docs/mitti-exporter-data#inspections)
  * [Inspection items](https://developer.mitti.com/docs/mitti-exporter-data#inspection-items)
  * [Templates](https://developer.mitti.com/docs/mitti-exporter-data#templates)
  * [Sites](https://developer.mitti.com/docs/mitti-exporter-data#sites)
  * [Users](https://developer.mitti.com/docs/mitti-exporter-data#users)
  * [Groups](https://developer.mitti.com/docs/mitti-exporter-data#groups)
  * [Group users](https://developer.mitti.com/docs/mitti-exporter-data#group-users)
  * [Activity log events](https://developer.mitti.com/docs/mitti-exporter-data#activity-log-events)
  * [Schedules](https://developer.mitti.com/docs/mitti-exporter-data#schedules)
  * [Schedule assignees](https://developer.mitti.com/docs/mitti-exporter-data#schedule-assignees)
  * [Schedule occurrences](https://developer.mitti.com/docs/mitti-exporter-data#schedule-occurrences)
  * [Actions](https://developer.mitti.com/docs/mitti-exporter-data#actions)
  * [Actions assignees](https://developer.mitti.com/docs/mitti-exporter-data#action-assignees)
  * [Issues](https://developer.mitti.com/docs/mitti-exporter-data#issues)

> The [Python version of Mitti Exporter](https://github.com/SafetyCulture/mitti-exporter/tree/v2) is no longer being maintained. We recommend downloading this latest version for faster exporting and additional data sets.

***

## Repository Structure

This is a monorepo containing both the CLI and the desktop UI:

```
.
├── cli/          # Go CLI tool (module: github.com/SafetyCulture/mitti-exporter)
├── ui/           # Wails v2 desktop app (React + Tailwind + shadcn/ui)
│   └── frontend/ # React frontend (pnpm)
├── go.work       # Go workspace linking cli/ and ui/
└── .github/      # CI workflows
```

The UI references the CLI as a local module via Go workspaces, so both always build against the same code.

## Development

### Prerequisites

* [Go 1.23+](https://golang.org/doc/install) (toolchain 1.24.4 is fetched automatically)
* [pnpm 10+](https://pnpm.io/installation) (for the UI frontend)
* [Node.js 20+](https://nodejs.org/) (LTS)
* [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation/) (for UI development)

### CLI

```bash
cd cli

# Unit tests
GOWORK=off go test ./...

# Build
GOWORK=off go build ./cmd/mitti-exporter
```

Use `GOWORK=off` for CLI-only work to avoid pulling in Wails/CGO dependencies.

When adding new columns in methods that implement `Columns() []string`, make sure they are added at the end to preserve CSV column order in export files.

### UI

```bash
cd ui

# Dev server with hot reload
wails dev

# Production build
wails build
```

### Integration Tests

SQL database integration tests require running databases:

```bash
cd cli
docker-compose up -d
make integration-tests
```

To run with local volume:

```bash
mkdir -p ~/docker-volume/mssql
docker-compose -f docker-compose-local-volume.yml up sqlserver
```

### Releasing

Releases are triggered by pushing a version tag. The CI pipeline builds CLI binaries (linux/amd64, darwin/amd64, darwin/arm64, windows/amd64) and desktop app bundles (macOS universal, Windows amd64), then publishes them as GitHub Release assets.

1. Checkout `main` and pull the latest changes.
2. Create a tag following [Semantic Versioning](https://semver.org):
   `git tag -a v3.1.0 -m "Release v3.1.0"`
   Acceptable formats: `v3.1.0`, `v3.1.0-alpha.1`, `v3.1.0-beta.1`
3. Push the tag: `git push origin v3.1.0`
4. Review and publish the [draft release](https://github.com/SafetyCulture/mitti-exporter/releases).
