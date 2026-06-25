#!/usr/bin/env bash
set -euo pipefail

REMOTE=${REMOTE:-makov@213.108.172.4}
REMOTE_DIR=${REMOTE_DIR:-/opt/max-bot}
ENV_FILE=${ENV_FILE:-.env}

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Env file '$ENV_FILE' not found. Create it from .env.example and fill production secrets." >&2
  exit 1
fi

ssh "$REMOTE" "mkdir -p '$REMOTE_DIR'"
rsync -az --delete \
  --exclude '.git' \
  --exclude '.env.local' \
  --exclude 'postgres_data' \
  ./ "$REMOTE:$REMOTE_DIR/"
scp "$ENV_FILE" "$REMOTE:$REMOTE_DIR/.env"
ssh "$REMOTE" "cd '$REMOTE_DIR' && docker compose up -d --build && docker compose ps"