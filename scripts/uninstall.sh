#!/usr/bin/env bash
# promptTranslate — Kaldırma Scripti
set -euo pipefail

BINARY_NAME="prompttranslate"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/prompttranslate"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_NAME="prompttranslate.service"
UDEV_FILE="/etc/udev/rules.d/80-prompttranslate-uinput.rules"

GREEN='\033[0;32m'; BLUE='\033[0;34m'; YELLOW='\033[1;33m'; RESET='\033[0m'; BOLD='\033[1m'
ok()   { echo -e "${GREEN}[OK]${RESET}   $*"; }
info() { echo -e "${BLUE}[INFO]${RESET} $*"; }
warn() { echo -e "${YELLOW}[WARN]${RESET} $*"; }

echo -e "${BOLD}promptTranslate — Kaldırma${RESET}"
echo

# Stop & disable service
info "Servis durduruluyor..."
systemctl --user stop "$SERVICE_NAME"  2>/dev/null && ok "Servis durduruldu." || true
systemctl --user disable "$SERVICE_NAME" 2>/dev/null && ok "Servis devre dışı bırakıldı." || true
rm -f "$SERVICE_DIR/$SERVICE_NAME"
systemctl --user daemon-reload
ok "Service dosyası silindi."

# Remove binary
if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
    rm -f "$INSTALL_DIR/$BINARY_NAME"
    ok "Binary silindi: $INSTALL_DIR/$BINARY_NAME"
fi

# Remove udev rule if present
if [[ -f "$UDEV_FILE" ]]; then
    sudo rm -f "$UDEV_FILE"
    sudo udevadm control --reload-rules
    ok "udev kuralı silindi: $UDEV_FILE"
fi

echo
warn "Config dizini korundu: $CONFIG_DIR"
warn "Silmek istiyorsanız: rm -rf $CONFIG_DIR"
echo
echo -e "${GREEN}${BOLD}Kaldırma tamamlandı.${RESET}"
