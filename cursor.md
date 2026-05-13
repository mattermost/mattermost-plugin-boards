# Agents

## Cursor Cloud specific instructions

> **Do NOT commit `AGENTS.md`.** It is generated at startup by the update script from this file (`cursor.md`). If you see `AGENTS.md` in `git status`, leave it unstaged.

### Overview

This is **Mattermost Boards** (formerly Focalboard) — a self-hosted project management plugin for Mattermost. It has two main build targets:

- **Go server plugin** (`server/`) — compiled with `CGO_ENABLED=0`
- **React/TypeScript webapp** (`webapp/`) — bundled with Webpack

### Running Mattermost + Boards locally

The development workflow uses Docker containers. On agent startup, Docker, PostgreSQL, and Mattermost containers should already be running (handled by the update script). To verify:

```sh
docker ps  # should show mattermost-server and mattermost-postgres
curl -s http://localhost:8065/api/v4/system/ping  # should return {"status":"OK"}
```

If containers are stopped (not removed), restart them:
```sh
sudo service docker start
docker start mattermost-postgres mattermost-server
```

### Building and deploying the plugin

```sh
MM_DEBUG=true make dist   # builds Go server + webapp bundle into dist/
MM_SERVICESETTINGS_SITEURL=http://localhost:8065 \
  MM_ADMIN_USERNAME=admin \
  MM_ADMIN_PASSWORD='Admin1234!' \
  ./build/bin/pluginctl deploy focalboard dist/boards-*.tar.gz
```

Using `MM_DEBUG=true` compiles only for the local OS/arch (much faster). Without it, all 5 platform binaries are built.

Note: `make deploy` re-runs `make dist` as a prerequisite. To skip the rebuild when the dist already exists, call `pluginctl deploy` directly as shown above.

### Mattermost admin credentials

- URL: `http://localhost:8065`
- Username: `admin`
- Password: `Admin1234!`
- Team: `dev-team`

### Lint, test, and type-check commands

See `Makefile` targets and `webapp/package.json` scripts. Key commands:

| Task | Command |
|------|---------|
| Webapp lint + stylelint | `cd webapp && npm run check` |
| Webapp type check | `cd webapp && npm run check-types` |
| Webapp unit tests | `cd webapp && npm run test` |
| Server lint | `cd server && golangci-lint run ./...` |
| Server unit tests | `FOCALBOARD_UNIT_TESTING=1 go test -race -v ./server/...` |
| Full plugin build | `MM_DEBUG=true make dist` |
| E2E tests | `make e2e` (requires Docker + built dist) |

### Browser automation with agent-browser

The `agent-browser` CLI is installed globally and can be used for headless browser automation. The agent-browser skill is at `.cursor/skills/agent-browser/SKILL.md`. Basic workflow:

```sh
agent-browser open http://localhost:8065/login
agent-browser snapshot -i          # see interactive elements
agent-browser fill @e9 "admin"     # fill username
agent-browser fill @e10 "Admin1234!"
agent-browser click @e8            # click login
agent-browser wait --load networkidle
agent-browser screenshot output.png
agent-browser close
```

### Gotchas

- The Boards API requires an `X-Requested-With: XMLHttpRequest` header for CSRF protection in addition to the `Authorization: Bearer` header.
- Three server test packages (`server/boards`, `server/integrationtests`, `server/services/store/sqlstore`) require a live PostgreSQL instance. They will fail in environments without one — this is expected. The remaining ~18 packages are pure unit tests and pass without a database.
- The `build/sync/` directory has its own `go.mod` and its tests may not compile with current Go versions due to API changes — this is a pre-existing repo issue, not an environment problem.
- Templates archive must be built before server compilation: `make templates-archive`. The `make dist` / `make server` targets handle this automatically.
