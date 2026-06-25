#!/usr/bin/env bash
# promptTranslate — Kurulum Scripti
# Kullanım: bash scripts/install.sh
set -euo pipefail

BINARY_NAME="prompttranslate"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/prompttranslate"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_NAME="prompttranslate.service"

# ─── Renkler ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'

info()  { echo -e "${BLUE}[INFO]${RESET}  $*"; }
ok()    { echo -e "${GREEN}[OK]${RESET}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
error() { echo -e "${RED}[ERROR]${RESET} $*" >&2; }
die()   { error "$*"; exit 1; }

echo -e "${BOLD}╔══════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║   promptTranslate — Kurulum Scripti     ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${RESET}"
echo

# ─── 1. Go kontrolü ───────────────────────────────────────────────────────────
info "Go kurulumu kontrol ediliyor..."
# Add common Go install locations to PATH
export PATH="$HOME/go/bin:/usr/local/go/bin:$PATH"
if ! command -v go &>/dev/null; then
    die "Go bulunamadı. Kurulum için README.md'e bakın (Go 1.22+)"
fi
GO_VER=$(go version | awk '{print $3}')
ok "Go bulundu: $GO_VER"

# ─── 2. Derleme ───────────────────────────────────────────────────────────────
info "Binary derleniyor..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

go mod tidy
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/prompttranslate/
ok "Derleme başarılı: $PROJECT_DIR/$BINARY_NAME"

# ─── 3. Binary kurulumu ───────────────────────────────────────────────────────
info "Binary $INSTALL_DIR dizinine kopyalanıyor..."
mkdir -p "$INSTALL_DIR"
cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"
ok "Binary kuruldu: $INSTALL_DIR/$BINARY_NAME"

# PATH kontrolü
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    warn "$INSTALL_DIR PATH'inizde değil."
    warn "~/.bashrc veya ~/.profile dosyanıza şunu ekleyin:"
    warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# ─── 4. input grubu ───────────────────────────────────────────────────────────
info "Klavye aygıtı erişimi kontrol ediliyor..."
if groups "$USER" | grep -qw "input"; then
    ok "Kullanıcı zaten 'input' grubunda."
else
    warn "Kullanıcı 'input' grubunda değil. Ekleniyor..."
    if sudo usermod -aG input "$USER"; then
        ok "Kullanıcı 'input' grubuna eklendi."
        warn "Değişikliğin geçerli olması için oturumu yeniden açın veya:"
        warn "  newgrp input"
    else
        warn "input grubuna eklenemedi. Daemon başlatılırken hata alabilirsiniz."
        warn "Manuel olarak: sudo usermod -aG input \$USER"
    fi
fi

# ─── 5. Wayland: ydotool udev kuralı ─────────────────────────────────────────
SESSION_TYPE="${XDG_SESSION_TYPE:-}"
if [[ "$SESSION_TYPE" == "wayland" ]] || [[ -n "${WAYLAND_DISPLAY:-}" ]]; then
    info "Wayland ortamı tespit edildi. ydotool udev kuralı yazılıyor..."
    UDEV_RULE='KERNEL=="uinput", GROUP="input", MODE="0660", OPTIONS+="static_node=uinput"'
    UDEV_FILE="/etc/udev/rules.d/80-prompttranslate-uinput.rules"
    if [[ ! -f "$UDEV_FILE" ]]; then
        echo "$UDEV_RULE" | sudo tee "$UDEV_FILE" > /dev/null
        sudo udevadm control --reload-rules
        sudo udevadm trigger
        ok "udev kuralı yazıldı: $UDEV_FILE"
    else
        ok "udev kuralı zaten mevcut: $UDEV_FILE"
    fi

    # ydotoold user service
    if ! systemctl --user is-active --quiet ydotoold 2>/dev/null; then
        info "ydotoold servisi başlatılıyor..."
        systemctl --user enable --now ydotoold 2>/dev/null || \
            warn "ydotoold servisi başlatılamadı. 'ydotoold &' ile manuel başlatın."
    else
        ok "ydotoold çalışıyor."
    fi
fi

# ─── 6. Konfigürasyon ─────────────────────────────────────────────────────────
info "Yapılandırma dizini oluşturuluyor..."
mkdir -p "$CONFIG_DIR"

if [[ ! -f "$CONFIG_DIR/config.toml" ]]; then
    cp "$PROJECT_DIR/configs/config.toml.example" "$CONFIG_DIR/config.toml"
    ok "Örnek config kopyalandı: $CONFIG_DIR/config.toml"
else
    ok "Config zaten mevcut: $CONFIG_DIR/config.toml"
fi

# ─── 7. API anahtarı rehberi ──────────────────────────────────────────────────
echo
echo -e "${BOLD}── API Anahtarı Kurulumu ────────────────────────────────────${RESET}"
echo -e "Config dosyasını düzenleyin:"
echo -e "  ${BLUE}nano $CONFIG_DIR/config.toml${RESET}"
echo
echo -e "Groq API anahtarı almak için:"
echo -e "  ${BLUE}https://console.groq.com/keys${RESET}"
echo
echo -e "Ya da ortam değişkeni kullanın:"
echo -e "  ${BLUE}echo 'export GROQ_API_KEY=\"gsk_...\"' >> ~/.bashrc${RESET}"
echo

# ─── 8. systemd user service ──────────────────────────────────────────────────
info "systemd user service yükleniyor..."
mkdir -p "$SERVICE_DIR"
cp "$PROJECT_DIR/systemd/$SERVICE_NAME" "$SERVICE_DIR/$SERVICE_NAME"
systemctl --user daemon-reload
ok "Service dosyası kuruldu: $SERVICE_DIR/$SERVICE_NAME"

echo
echo -e "${BOLD}── Sonraki Adımlar ──────────────────────────────────────────${RESET}"
echo -e "1. API anahtarınızı ayarlayın (yukarıya bakın)"
echo -e "2. Bağımlılıkları kontrol edin:"
echo -e "   ${BLUE}$BINARY_NAME --check${RESET}"
echo -e "3. Servisi başlatın:"
echo -e "   ${BLUE}systemctl --user enable --now $SERVICE_NAME${RESET}"
echo -e "4. Durumu kontrol edin:"
echo -e "   ${BLUE}systemctl --user status $SERVICE_NAME${RESET}"
echo -e "5. Logları izleyin:"
echo -e "   ${BLUE}journalctl --user -u $SERVICE_NAME -f${RESET}"
echo
echo -e "${GREEN}${BOLD}Kurulum tamamlandı!${RESET}"
echo -e "Kısayollar: ${BOLD}Alt+Space${RESET} (TR→EN yaz) | ${BOLD}Alt+Shift+Space${RESET} (EN→TR oku)"
