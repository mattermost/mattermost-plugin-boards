#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if [[ -f .cursor/cursor.md ]]; then
    cp .cursor/cursor.md .cursor/AGENTS.md
fi

if [[ -f /proc/sys/kernel/apparmor_restrict_unprivileged_userns ]]; then
    sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0 >/dev/null || true
fi

apply_docker_socket_acl() {
    if [[ -S /var/run/docker.sock ]]; then
        sudo chgrp docker /var/run/docker.sock || true
        sudo chmod g+rw /var/run/docker.sock || true
        sudo setfacl -m "u:${USER}:rw" /var/run/docker.sock || true
    fi
}

if ! docker info >/dev/null 2>&1; then
    if command -v service >/dev/null 2>&1; then
        sudo sh -c 'service docker start >/tmp/docker-service-start.log 2>&1' || true
    fi

    if ! pgrep -x dockerd >/dev/null 2>&1; then
        sudo rm -f /var/run/docker.pid
        sudo sh -c 'nohup dockerd --host=unix:///var/run/docker.sock >/tmp/dockerd.log 2>&1 &'
    fi
fi

for _ in $(seq 1 60); do
    apply_docker_socket_acl
    if docker info >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

if ! docker info >/dev/null 2>&1; then
    echo "Docker did not become ready within 60 seconds." >&2
    echo "docker service log:" >&2
    tail -200 /tmp/docker-service-start.log 2>/dev/null || true
    echo "dockerd log:" >&2
    tail -200 /tmp/dockerd.log 2>/dev/null || true
    exit 1
fi

load_image_archive() {
    local image_ref="$1"
    local archive="$2"

    if docker image inspect "$image_ref" >/dev/null 2>&1; then
        return
    fi

    if [[ -f "$archive" ]]; then
        docker load -i "$archive"
        return
    fi

    echo "Preloaded archive not found for $image_ref; pulling from registry." >&2
    docker pull "$image_ref"
}

if [[ "${CLOUD_AGENT_SKIP_IMAGE_LOAD:-}" != "1" ]]; then
    load_image_archive "${MATTERMOST_IMAGE:-mattermostdevelopment/mattermost-enterprise-edition}:${MATTERMOST_IMAGE_TAG:-master}" /opt/cursor-prepulled/mattermost-enterprise-edition.tar
    load_image_archive "${POSTGRES_IMAGE:-postgres}:${POSTGRES_IMAGE_TAG:-16-alpine}" /opt/cursor-prepulled/postgres.tar
fi

echo "Cloud agent start complete. Docker is ready."
