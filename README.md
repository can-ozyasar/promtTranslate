<div align="center">

# 🌐 promptTranslate

**Linux için Global Kısayollu TR↔EN Çeviri Daemon'ı**

*Terminal akışını bozmadan, tek tuşla, gerçek zamanlı çeviri.*

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square&logo=linux)](https://kernel.org)
[![Session](https://img.shields.io/badge/Session-X11%20%7C%20Wayland-blue?style=flat-square)]()

</div>

---

## Neden Böyle Bir Araca İhtiyaç Vardı?

Türkçe konuşan bir yazılımcı olarak her gün şu döngüyü yaşıyorsunuz:

1. Terminalde veya editörde çalışıyorsunuz.
2. **Bir İngilizce prompt yazmak** istiyorsunuz — tarayıcıya geçiyorsunuz, Google Translate veya DeepL açıyorsunuz, Türkçeyi yazıyorsunuz, çeviriyi kopyalıyorsunuz, terminale dönüyorsunuz, yapıştırıyorsunuz.
3. **Bir İngilizce metni anlamak** istiyorsunuz — yine tarayıcıya, yine yapıştırma, yine geri.

Her seferinde **5–10 saniye** ve **bağlam kaybı**. Gün içinde onlarca kez yapıyorsunuz. Bu, odaklanmayı kıran, verimliliği düşüren friksiyon'dur.

**promptTranslate bu friksiyonu sıfıra indiriyor:**

- `Alt+Space` → Türkçe yazın → aktif terminale İngilizce **anında enjekte edilir**
- `Alt+Shift+Space` → Seçili İngilizce metin → Türkçe çeviri **sistem bildirimi olarak gelir**

Tarayıcı yok. Fare yok. Bağlam kaybı yok.

---

## Özellikler

| Özellik | Detay |
|---------|-------|
| ⚡ **Hız** | Groq Llama 3.1 ile ~150–300ms ortalama gecikme |
| 🖥️ **Çift ortam** | X11 ve Wayland otomatik algılama |
| 🧠 **LRU Cache** | Son 50 çeviri bellekte — tekrar sorulan anında yanıtlanır |
| 🔄 **Akıllı retry** | API hatalarında 3x exponential backoff |
| 🔔 **Bildirim** | `notify-send` ile yerel sistem bildirimi |
| 📋 **Pano desteği** | Çeviri hem bildirimde hem panoda |
| 🛡️ **Güvenlik** | systemd hardening, secrets dosyaya yazılmaz |
| 📦 **Tek binary** | CGO yok, 5.4MB statik ELF, her Linux dağıtımında çalışır |
| 💾 **Düşük kaynak** | ~3MB RAM, idle'da %0 CPU |

---

## Çalışma Akışı

### Yazma Modu: TR → EN (Alt+Space)

```
Kullanıcı Alt+Space basar
    ↓
rofi/wofi dmenu ekranda belirir
    ↓
"Şu kodu bellek açısından optimize et" yazar
    ↓
Groq API (Llama 3.1-8b-instant, ~200ms)
    ↓
"Optimize this code for memory efficiency"
    ↓
xdotool/ydotool → aktif terminale tuş enjeksiyonu
```

### Okuma Modu: EN → TR (Alt+Shift+Space)

```
Kullanıcı terminaldeki metni fareyle seçer
    ↓
Alt+Shift+Space basar
    ↓
Primary selection / pano okunur (xclip / wl-paste)
    ↓
Groq API çevirir
    ↓
notify-send bildirimi + panoya kopyalanır
```

---

## Mimari

```
┌─────────────────────────────────────────────────────────────┐
│                  prompttranslate daemon                      │
│                                                             │
│  ┌────────────┐    ┌────────────┐    ┌──────────────────┐  │
│  │   Hotkey   │───▶│   Input    │───▶│   Translator     │  │
│  │  Listener  │    │  Manager   │    │  Engine          │  │
│  │ (evdev)    │    │ rofi/clip  │    │  Groq/DeepL      │  │
│  └────────────┘    └────────────┘    │  + LRU Cache     │  │
│                                      └────────┬─────────┘  │
│                                               │             │
│                               ┌───────────────┴──────────┐ │
│                               ▼                          ▼ │
│                      ┌──────────────┐        ┌──────────┐  │
│                      │   Injector   │        │  Notify  │  │
│                      │ xdotool /    │        │  -send   │  │
│                      │ ydotool      │        │          │  │
│                      └──────────────┘        └──────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Modül Açıklamaları

| Modül | Paket | Görev |
|-------|-------|-------|
| **Hotkey Listener** | `internal/hotkey` | `/dev/input/event*` aygıtlarını doğrudan okur (evdev). Root gerektirmez — `input` grubu yeterli. |
| **Input Manager** | `internal/input` | rofi/wofi dmenu subprocess'i yönetir; panodan/primary selection'dan metin çeker. |
### 🖥️ Arayüz & Tepsi Simgesi (Tray Icon)
Arka plan servisi çalıştığında, GNOME veya diğer masaüstü ortamlarının üst panelinde bir **Tepsi Simgesi (Sözlük İkonu)** belirir:
- **🟢/🔴 Durum Göstergesi:** Servisin anlık olarak çalışıp çalışmadığını gösterir.
- **▶ Başlat / ⏹ Durdur:** Servisi tek tıkla açıp kapatabilirsiniz.
- Bilgisayar her açıldığında tepsi simgesi otomatik olarak başlatılır (`~/.config/autostart/prompttranslate-tray.desktop`).

### 🛠️ Nasıl Çalışır?

#### 1. X11 Ortamı
X11 üzerinde `xdotool` kullanılarak çevrilen metin *anında* aktif terminale veya dökümana klavye simülasyonu ile enjekte edilir. Pürüzsüz ve gerçek zamanlıdır.

#### 2. Wayland Ortamı (GNOME)
Wayland'ın (özellikle GNOME) katı güvenlik modeli nedeniyle klavye simülasyonu (`ydotool`) stabil çalışmamaktadır. Bu yüzden Wayland tespit edildiğinde sistem otomatik olarak **Pano (Clipboard) Moduna** geçer:
1. Çevrilen metin anında panoya (`wl-copy`) kopyalanır.
2. Sağ üstte "Çeviri Hazır! 📋" şeklinde bir bildirim (`notify-send`) çıkar.
3. Siz sadece `Ctrl+V` yaparak metni istediğiniz yere yapıştırırsınız. Hata payı %0'dır.iyonu. Modifier temizleme ve gecikme yönetimi dahil. |
| **Notify** | `internal/notify` | `notify-send` sarmalayıcı; aciliyet, süre ve ikon desteği. |
| **Config** | `internal/config` | TOML + XDG_CONFIG_HOME + env var override. Sıfır bağımlılık, varsayılanlarla çalışır. |
| **Env Detect** | `internal/env` | `$WAYLAND_DISPLAY` / `$DISPLAY` ile ortam tespiti; araç bağımlılık kontrolü. |

---

## Kurulum

### Gereksinimler

**Go 1.22+:**
```bash
# Hızlı kurulum (kullanıcı dizinine, sudo gerekmez)
wget -q https://go.dev/dl/go1.22.4.linux-amd64.tar.gz -O /tmp/go.tar.gz
tar -C "$HOME" -xzf /tmp/go.tar.gz
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
go version  # go version go1.22.4 linux/amd64
```

**Sistem araçl```bash
# Ubuntu/Debian için eksik bağımlılıkları yükleyin
sudo apt install -y rofi wl-clipboard xclip ydotool

# Wayland ve GNOME için üst menü (Tepsi Simgesi / Tray Icon) bağımlılıkları:
sudo apt install -y python3-gi gir1.2-ayatanaappindicator3-0.1 gir1.2-appindicator3-0.1

# 2. Kurulum betiğini çalıştırın
```

**Sistem araçları (X11 için):**
```bash
sudo apt install rofi xclip xdotool libnotify-bin
```

### Ücretsiz Groq API Key

1. [console.groq.com](https://console.groq.com) → Sign up (Google/GitHub, kredi kartı yok)
2. **API Keys** → **Create API Key** → Kopyala (`gsk_...` ile başlar)
3. Terminale ekle:
```bash
echo 'export GROQ_API_KEY="gsk_ANAHTARINIZ"' >> ~/.bashrc
source ~/.bashrc
```

> **Ücretsiz limit:** Dakikada 30 istek, günde ~14.000 token.
> Günlük kullanım için fazlasıyla yeterli (1 çeviri ≈ 50–100 token).

### Kurulum Scripti

```bash
git clone https://github.com/canoz/promttranslate.git
cd promttranslate
bash scripts/install.sh
```

Script otomatik olarak şunları yapar:
- ✅ Binary'yi derler ve `~/.local/bin/` içine kurar
- ✅ Kullanıcıyı `input` grubuna ekler (evdev erişimi)
- ✅ Wayland'da `ydotool` için udev kuralı yazar
- ✅ `~/.config/prompttranslate/config.toml` oluşturur
- ✅ systemd user service'i yükler ve etkinleştirir

### Bağımlılık Kontrolü

```bash
prompttranslate --check
```

Örnek çıktı:
```
promptTranslate — dependency check (wayland)
──────────────────────────────────────────────────
✅  All required dependencies found.
```

### Servisi Başlat

```bash
# Oturum açışında otomatik başlatılır
systemctl --user enable --now prompttranslate.service

# Durum
systemctl --user status prompttranslate.service

# Gerçek zamanlı log
journalctl --user -u prompttranslate.service -f
```

---

## Kullanım

| Kısayol | Mod | Eylem |
|---------|-----|-------|
| `Alt+Space` | **Yazma** | rofi açılır → TR yazın → aktif pencereye EN enjekte edilir |
| `Alt+Shift+Space` | **Okuma** | Seçili/kopyalı EN metin → Türkçe bildirim + pano |
| `Alt+Shift+C` | **Yeniden yükle** | Config yenileme bildirimi |

### Örnek Senaryolar

**1. AI'a İngilizce prompt göndermek:**
```
Terminaldeyken Alt+Space → "bu fonksiyonun karmaşıklığını azalt" →
→ "Reduce the complexity of this function" aktif terminale yazılır
```

**2. İngilizce hata mesajını anlamak:**
```
"cannot use x (type int) as type string" metnini fareyle seç →
Alt+Shift+Space → "x'i (int türü) string türü olarak kullanamazsınız" bildirimi gelir
```

**3. İngilizce dökümantasyon okumak:**
```
Paragrafı kopyala → Alt+Shift+Space → Türkçe özet bildirim gelir
```

---

## Yapılandırma

`~/.config/prompttranslate/config.toml`:

```toml
[translator]
provider    = "groq"   # groq | deepl
cache_size  = 50       # LRU cache boyutu
timeout_sec = 10
max_retries = 3

[groq]
model    = "llama-3.1-8b-instant"  # en hızlı model
api_key  = ""                       # ya da GROQ_API_KEY env var

[hotkeys]
write_mode = "alt+space"
read_mode  = "alt+shift+space"

[display]
launcher = "rofi"   # rofi | wofi
theme    = "prompttranslate"

[injection]
keystroke_delay_ms = 12  # terminale göre ayarlayın (5-30)
```

Tüm seçenekler için: [`configs/config.toml.example`](configs/config.toml.example)

---

## Geliştirme

```bash
make build       # CGO_ENABLED=0, statik binary
make check       # bağımlılık kontrolü
make test-write  # yazma modunu tek seferlik dene (rofi açılır)
make test-read   # okuma modunu tek seferlik dene
make test        # unit testler (go test -race)
make vet         # statik analiz
make help        # tüm hedefler
```

### Proje Yapısı

```
promtTranslate/
├── cmd/prompttranslate/
│   └── main.go              # giriş noktası, flag parsing, orkestrasyon
├── internal/
│   ├── config/config.go     # TOML yükleyici + doğrulama + XDG
│   ├── env/detect.go        # ortam tespiti + bağımlılık tarayıcı
│   ├── hotkey/
│   │   ├── keymap.go        # Linux evdev KEY_* sabitleri
│   │   └── listener.go      # pure-Go evdev okuyucu goroutine
│   ├── input/
│   │   ├── rofi.go          # rofi/wofi subprocess + inline tema
│   │   └── clipboard.go     # xclip/wl-paste + WriteClipboard
│   ├── translator/
│   │   ├── engine.go        # Translator interface + LRU cache
│   │   ├── groq.go          # Groq REST client (keep-alive pool)
│   │   └── deepl.go         # DeepL REST client (free/pro)
│   ├── injector/
│   │   ├── injector.go      # Injector interface
│   │   ├── xdotool.go       # X11 enjektörü
│   │   └── ydotool.go       # Wayland enjektörü
│   └── notify/
│       └── notify.go        # notify-send sarmalayıcı
├── configs/
│   └── config.toml.example  # örnek yapılandırma
├── systemd/
│   └── prompttranslate.service
├── scripts/
│   ├── install.sh
│   └── uninstall.sh
├── Makefile
└── go.mod
```

---

## Teknik Detaylar

### Neden evdev?

Ubuntu'nun kısayol yöneticisi (`gsettings`) daemon süreçlerinden tetiklenemiyor. `evdev` ile `/dev/input/event*` aygıtlarını doğrudan okuyarak, X11/Wayland fark etmeksizin global kısayol dinleme gerçekleştiriyoruz. `input` grubu üyeliği ile root yetkisi **gerekmez**.

### Neden `xdotool --clearmodifiers`?

`Alt+Space` kısayolu tetiklendiğinde `Alt` tuşu hâlâ basılı. Temizlemeden enjeksiyon yapılırsa terminal emülatörü tuşları yanlış yorumlar. `--clearmodifiers` tüm modifier tuşları serbest bırakır, `--delay 12` ise çok hızlı enjeksiyonun terminal tarafından kaçırılmasını önler.

### Neden HTTP keep-alive?

Her çeviri isteği için yeni TCP+TLS bağlantısı açmak ~50–200ms ek gecikme yaratır. `MaxIdleConnsPerHost: 10` ile bağlantı havuzu tutulur; ilk istek sonrası her çeviri soğuk başlatma maliyeti olmadan gider.

### Neden Go?

- **Tek statik binary** → dağıtım kolaylığı, paket bağımlılığı yok
- **Goroutine** → evdev okuyucu bloklamadan çalışır, çeviri asenkron
- **~3MB RAM** → idle sistemde neredeyse görünmez
- **CGO yok** → cross-compilation, container uyumlu

---

## Sorun Giderme

### Kısayol çalışmıyor
```bash
# input grubunu kontrol et
groups $USER | grep -c input

# Grup yoksa (yeniden oturum açmak gerekir):
sudo usermod -aG input $USER
newgrp input
```

### Wayland'da metin enjekte edilmiyor
```bash
# ydotoold çalışıyor mu?
systemctl --user status ydotoold

# Değilse:
systemctl --user enable --now ydotoold
```

### rofi açılmıyor
```bash
# Kurulu mu?
which rofi || sudo apt install rofi

# Wayland'dayken sorun varsa wofi dene:
# config.toml → [display] launcher = "wofi"
sudo apt install wofi
```

### API hata veriyor
```bash
# Key geçerli mi?
curl -s https://api.groq.com/openai/v1/models \
  -H "Authorization: Bearer $GROQ_API_KEY" | head -c 100

# Log'lara bak:
journalctl --user -u prompttranslate.service -n 50
```

---

## Kaldırma

```bash
bash scripts/uninstall.sh
```

---

## Lisans

[MIT](LICENSE) © 2024 canoz

---

<div align="center">

**Friksiyonsuz çeviri. Terminal odaklanması.**

*Tarayıcıya geçme. Yapıştırma. Geri dön. Artık yok.*

</div>
