#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# go-dbms server bootstrap
#
# Prerequisites (must already be true before running this):
#   1. The server has internet access (netplan/DHCP or static IP
#      already configured and verified — this script cannot fix
#      networking, since it needs the internet itself to run).
#   2. This repo is cloned on the server and you're running the
#      script from inside it:
#        git clone https://github.com/JpUnique/go-dbms.git
#        cd go-dbms
#        chmod +x scripts/server-setup.sh
#        ./scripts/server-setup.sh
#
# Safe to re-run: every step checks whether it already happened
# and skips it if so. If it stops asking you to edit .env, just
# fill that in and run it again — it picks up where it left off.
# ============================================================

REACT_REPO_URL="https://github.com/JpUnique/react-dbms.git"
FRONTEND_DIR="../react-dbms"
SERVICE_USER="$(whoami)"

log()  { printf '\n\033[1;32m==> %s\033[0m\n' "$1"; }
warn() { printf '\033[1;33m!! %s\033[0m\n' "$1"; }

# ---- 0. sanity: internet must already be up ----
log "Checking internet connectivity"
if ! ping -c1 -W3 8.8.8.8 >/dev/null 2>&1; then
  echo "No internet connectivity detected. Configure netplan first, verify with 'ping 8.8.8.8', then re-run this script." >&2
  exit 1
fi

# ---- 1. base packages ----
log "Updating apt and installing base packages"
sudo apt-get update -y
sudo apt-get install -y git curl ufw ca-certificates gnupg

# ---- 2. Docker ----
if ! command -v docker >/dev/null 2>&1; then
  log "Installing Docker"
  curl -fsSL https://get.docker.com | sudo sh
  sudo usermod -aG docker "$SERVICE_USER"
  warn "Added $SERVICE_USER to the docker group — log out/in later for non-sudo docker to work. This script uses 'sudo docker' throughout, so it works regardless."
else
  log "Docker already installed, skipping"
fi

# ---- 3. Node.js + pnpm ----
if ! command -v node >/dev/null 2>&1; then
  log "Installing Node.js 20.x"
  curl -fsSL https://deb.nodesource.com/setup_20.x | sudo bash -
  sudo apt-get install -y nodejs
else
  log "Node.js already installed ($(node --version)), skipping"
fi

if ! command -v pnpm >/dev/null 2>&1; then
  log "Enabling corepack / pnpm"
  sudo corepack enable
  corepack prepare pnpm@8.10.0 --activate
else
  log "pnpm already installed ($(pnpm --version)), skipping"
fi

# ---- 4. Tailscale ----
if ! command -v tailscale >/dev/null 2>&1; then
  log "Installing Tailscale"
  curl -fsSL https://tailscale.com/install.sh | sh
else
  log "Tailscale already installed, skipping"
fi

if ! sudo tailscale status >/dev/null 2>&1; then
  log "Bringing up Tailscale"
  if [ -n "${TAILSCALE_AUTHKEY:-}" ]; then
    sudo tailscale up --authkey "$TAILSCALE_AUTHKEY"
  else
    warn "No TAILSCALE_AUTHKEY set — open the login URL this prints in a browser to authorize this machine."
    sudo tailscale up
  fi
else
  log "Tailscale already connected, skipping"
fi

TAILSCALE_IP="$(tailscale ip -4)"
log "Tailscale IP: $TAILSCALE_IP"

# ---- 5. Firewall ----
log "Configuring ufw (SSH + Tailscale only — app ports are NOT exposed to the LAN/internet)"
sudo ufw allow 22/tcp
sudo ufw allow in on tailscale0
sudo ufw --force enable

# ---- 6. Backend .env ----
if [ ! -f .env ]; then
  log "No .env found — creating from .env.example"
  cp .env.example .env
  warn "Edit .env now and fill in every CHANGE_ME value (JWT secrets via 'openssl rand -hex 32', admin credentials), then re-run this script."
  exit 0
fi

if grep -q "CHANGE_ME" .env; then
  warn ".env still has CHANGE_ME placeholders. Edit .env and fill them in, then re-run this script."
  exit 1
fi

# ---- 7. Backend containers ----
log "Starting backend containers (postgres, minio, clamd, backend)"
sudo docker compose up -d --build

log "Container status:"
sudo docker ps -a

# ---- 8. Frontend source ----
if [ ! -d "$FRONTEND_DIR" ]; then
  log "Cloning react-dbms"
  git clone "$REACT_REPO_URL" "$FRONTEND_DIR"
fi

log "Building frontend against Tailscale IP $TAILSCALE_IP"
(
  cd "$FRONTEND_DIR"
  echo "VITE_API_URL=http://$TAILSCALE_IP:4000/dbms/v1" > .env.production
  pnpm install
  pnpm build
)

# ---- 9. Frontend systemd service ----
log "Installing systemd service for the frontend (survives reboots/crashes)"
FRONTEND_ABS_PATH="$(cd "$FRONTEND_DIR" && pwd)"
sudo tee /etc/systemd/system/react-dbms.service >/dev/null <<EOF
[Unit]
Description=React DBMS Frontend (vite preview)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
WorkingDirectory=$FRONTEND_ABS_PATH
ExecStart=$(command -v npx) vite preview --host 0.0.0.0 --port 3000
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now react-dbms

log "Done"
echo "Backend:  http://$TAILSCALE_IP:4000/dbms/v1"
echo "Frontend: http://$TAILSCALE_IP:3000"
echo
echo "Install Tailscale on each team member's device and join this same tailnet"
echo "to reach the app from the office or anywhere else."
