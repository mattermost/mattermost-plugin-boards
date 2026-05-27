# Agents

## Build commands

- `make apply` — Propagate plugin manifest into server/ and webapp/
- `MM_DEBUG=true make dist` — Build plugin bundle (single platform, faster for local dev)
- `make check-style` — Run all linters (Go + webapp)
- `make test` — Run all tests (Go + webapp)
- `make dist` — Build all platform assets

## Cursor Cloud Agents

- Cloud-agent environment files live in `.cursor/`.
- `.cursor/cursor.md` has cloud-only instructions for starting Mattermost with Docker and deploying this plugin.
- `.cursor/AGENTS.md` is generated from `.cursor/cursor.md` during cloud-agent startup and should not be committed.
