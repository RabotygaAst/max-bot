# Deploy to Debian server

Use `scripts/deploy-debian.sh` to copy the current repository to a Debian host and start the bot with Docker Compose.

## Prerequisites on the server

- Docker Engine with the Compose plugin (`docker compose`).
- SSH access to the deployment user.
- A completed production `.env` file based on `.env.example`.

## Required production values

At minimum, replace placeholders in the env file before deploying:

```dotenv
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=<real MAX bot token>
WEBHOOK_SECRET=<long random webhook secret>
ONEC_BASE_URL=<published or internal 1C HTTP service URL>
ONEC_TOKEN=<1C integration token>
INTERNAL_API_TOKEN=<long random internal notification token>
```

## Deploy command

From the repository root:

```bash
REMOTE=makov@213.108.172.4 REMOTE_DIR=/opt/max-bot ENV_FILE=.env ./scripts/deploy-debian.sh
```

The script creates the remote directory, syncs the repository, uploads the selected env file as `.env`, rebuilds the Docker image, starts PostgreSQL, mock 1C, and the bot, then prints `docker compose ps`.

If the production 1C endpoint is used, set `ONEC_BASE_URL` to that endpoint instead of the local mock URL.

## Troubleshooting: Docker cannot resolve Docker Hub

If `docker compose up -d --build` fails with an error like `lookup registry-1.docker.io ... i/o timeout`, the server cannot resolve or reach Docker Hub from the Docker daemon. This is usually a DNS, firewall, proxy, or outbound network issue on the server, not an application error.

Check DNS from the host:

```bash
getent hosts registry-1.docker.io
nslookup registry-1.docker.io
curl -I https://registry-1.docker.io/v2/
```

If DNS lookup times out, configure Docker daemon DNS explicitly:

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json >/dev/null <<'JSON'
{
  "dns": ["1.1.1.1", "8.8.8.8"]
}
JSON
sudo systemctl restart docker
```

Then retry the deploy:

```bash
cd /opt/max-bot/max-bot
sudo docker compose pull
sudo docker compose up -d --build
```

If the server is in a closed corporate network, allow outbound HTTPS to Docker Hub or configure the Docker daemon HTTP/HTTPS proxy before retrying.

## Production compose without mock 1C

For a server connected to a real 1C endpoint, use the production compose file. It does not start or pull `mockserver/mockserver`:

```bash
cd /opt/max-bot/max-bot
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d --build
```

The default `docker-compose.yml` is still useful for local smoke tests because it includes `mock-onec`.

## Troubleshooting: host DNS still points to an unavailable resolver

Docker image pulls are resolved by the host/Docker daemon. If `curl -I https://registry-1.docker.io/v2/` still says `Could not resolve host` and Docker errors still mention an old resolver such as `77.88.8.8`, fix the host resolver first.

Check the active resolver:

```bash
resolvectl status
cat /etc/resolv.conf
ip route
```

If the server uses `systemd-resolved`, configure reachable DNS servers:

```bash
sudo mkdir -p /etc/systemd/resolved.conf.d
sudo tee /etc/systemd/resolved.conf.d/dns.conf >/dev/null <<'EOF_DNS'
[Resolve]
DNS=1.1.1.1 8.8.8.8
FallbackDNS=77.88.8.8
EOF_DNS
sudo systemctl restart systemd-resolved
sudo resolvectl flush-caches
```

Then re-check DNS and Docker Hub access:

```bash
getent hosts registry-1.docker.io
curl -I https://registry-1.docker.io/v2/
```

If `ping -c 3 1.1.1.1` and HTTPS requests to Docker Hub both fail, the server likely has no outbound internet route or the network blocks egress. In that case, use the organization's DNS/proxy/mirror or allow outbound TCP 443 to Docker Hub before running Compose again.
