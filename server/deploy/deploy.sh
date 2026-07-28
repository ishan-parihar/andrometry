#!/bin/bash
# Andrometry Server Deployment Script
# §28 Server Deployment Example
#
# Usage: sudo bash deploy.sh

set -euo pipefail

APP_NAME="andrometry"
APP_USER="andrometry"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/andrometry"
CONFIG_DIR="/etc/andrometry"

echo "=== Andrometry Server Deployment ==="

# Create user
if ! id "$APP_USER" &>/dev/null; then
    echo "Creating user: $APP_USER"
    useradd --system --no-create-home --shell /bin/false "$APP_USER"
fi

# Create directories
echo "Creating directories..."
mkdir -p "$DATA_DIR" "$CONFIG_DIR"
chown "$APP_USER:$APP_USER" "$DATA_DIR"
chmod 750 "$DATA_DIR"

# Copy binary
echo "Installing binary..."
cp "$(dirname "$0")/../cmd/andrometry/andrometry" "$INSTALL_DIR/$APP_NAME"
chown root:root "$INSTALL_DIR/$APP_NAME"
chmod 755 "$INSTALL_DIR/$APP_NAME"

# Copy config
echo "Installing configuration..."
cp "$(dirname "$0")/../config.toml" "$CONFIG_DIR/config.toml"
chown root:root "$CONFIG_DIR/config.toml"
chmod 640 "$CONFIG_DIR/config.toml"

# Copy systemd service
echo "Installing systemd service..."
cp "$(dirname "$0")/andrometry.service" /etc/systemd/system/
systemctl daemon-reload

# Configure firewall
echo "Configuring firewall..."
if command -v ufw &>/dev/null; then
    ufw allow 443/tcp comment "Andrometry HTTPS"
    ufw allow 8420/tcp comment "Andrometry Relay (internal)"
fi

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps:"
echo "  1. Bootstrap a token:  sudo $INSTALL_DIR/$APP_NAME --db $DATA_DIR/$APP_NAME.db --bootstrap-token YOUR_SECRET_TOKEN"
echo "  2. Start the service:  sudo systemctl start $APP_NAME"
echo "  3. Enable on boot:     sudo systemctl enable $APP_NAME"
echo "  4. Check status:       sudo systemctl status $APP_NAME"
echo "  5. View logs:          sudo journalctl -u $APP_NAME -f"
