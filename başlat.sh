#!/usr/bin/env bash
# promptTranslate — Tek Komutlu Başlatıcı
set -euo pipefail

BINARY="$HOME/.local/bin/prompttranslate"

# ── API Key kontrol ────────────────────────────────────────────────────────────
if [[ -z "${GROQ_API_KEY:-}" ]]; then
    echo ""
    echo "🔑 GROQ_API_KEY bulunamadı."
    echo ""
    echo "  1. https://console.groq.com/keys → Sign up (ücretsiz, kart yok)"
    echo "  2. 'Create API Key' → kopyala"
    echo "  3. Aşağıya yapıştır:"
    echo ""
    read -rp "  API Key: " key
    if [[ -z "$key" ]]; then
        echo "❌ API key girilmedi. Çıkılıyor."
        exit 1
    fi
    export GROQ_API_KEY="$key"
    echo "export GROQ_API_KEY=\"$key\"" >> "$HOME/.bashrc"
    echo "✅ API key .bashrc'e kaydedildi."
fi

# ── Binary var mı? ─────────────────────────────────────────────────────────────
if [[ ! -f "$BINARY" ]]; then
    echo "❌ Binary bulunamadı: $BINARY"
    echo "   Önce kurulum yapın: bash scripts/install.sh"
    exit 1
fi

# ── Servisi durdur (varsa eski oturum) ────────────────────────────────────────
systemctl --user stop prompttranslate.service 2>/dev/null || true

# ── input grubunu aktif et ve başlat ──────────────────────────────────────────
echo ""
echo "✅ Başlatılıyor..."
echo "   Kısayollar: Alt+Space (TR→EN) | Alt+Shift+Space (EN→TR oku)"
echo "   Durdurmak: Ctrl+C"
echo ""

# sg komutu ile mevcut terminalde input grubu aktif edilir (logout gerekmez)
exec sg input -c "GROQ_API_KEY='$GROQ_API_KEY' $BINARY"
