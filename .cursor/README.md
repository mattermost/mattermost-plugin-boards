# Cursor Cloud Agent Environment

This directory defines the checked-in environment for Cursor Cloud Agents.

- `environment.json` selects the Dockerfile build and declares the Mattermost port.
- `Dockerfile` installs Go, Node.js, Docker-in-Docker support, Docker Compose, AWS CLI, `agent-browser`, Chrome runtime libraries, and cached Mattermost/Postgres image archives.
- `scripts/cloud-agent-install.sh` hydrates Go and webapp dependencies.
- `scripts/cloud-agent-start.sh` starts `dockerd`, fixes socket permissions, loads cached images, and materializes `.cursor/AGENTS.md`.
- `cursor.md` contains cloud-only instructions for running Mattermost and deploying this plugin.

`.cursor/AGENTS.md` is generated at cloud-agent startup from `cursor.md` and should not be committed.

## Validation

From the repository root:

```bash
python3 -m json.tool .cursor/environment.json >/dev/null
bash -n .cursor/scripts/cloud-agent-install.sh && bash -n .cursor/scripts/cloud-agent-start.sh
docker build --check -f .cursor/Dockerfile .cursor
docker build -f .cursor/Dockerfile .cursor/
```

The Dockerfile intentionally fetches `mattermostdevelopment/mattermost-enterprise-edition:master` and `postgres:16-alpine` during image build so cloud-agent startup can load them locally before running Mattermost. The Mattermost development image is pinned to `linux/amd64` because the `master` tag does not publish an arm64 image. Browser assets are installed during amd64 image builds; local arm64 builds validate the CLI but skip the browser download because Chrome for Testing does not publish Linux arm64 builds.
