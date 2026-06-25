BINARY     := prompttranslate
CMD_PATH   := ./cmd/prompttranslate
INSTALL_DIR := $(HOME)/.local/bin
BUILD_FLAGS := -ldflags="-s -w"

# Auto-detect Go binary (handles ~/go/bin, /usr/local/go/bin, and PATH)
GO := $(shell which go 2>/dev/null || echo $(HOME)/go/bin/go)
export PATH := $(HOME)/go/bin:$(PATH)
export GOPATH := $(HOME)/.gopath

.PHONY: all build install uninstall check test lint clean help

all: build

## build: Derleme (CGO olmadan, tek statik binary)
build:
	CGO_ENABLED=0 $(GO) build $(BUILD_FLAGS) -o $(BINARY) $(CMD_PATH)
	@echo "✅ Derlendi: $(BINARY)"

## install: Derleme + tam kurulum
install:
	@bash scripts/install.sh

## uninstall: Temiz kaldırma
uninstall:
	@bash scripts/uninstall.sh

## check: Bağımlılık kontrolü
check: build
	./$(BINARY) --check

## test-write: Bir kez yazma modunu test et (rofi açılır)
test-write: build
	./$(BINARY) --once-write

## test-read: Bir kez okuma modunu test et (panonuzu okur)
test-read: build
	./$(BINARY) --once-read

## test: Unit testleri çalıştır
test:
	$(GO) test -v -race ./...

## vet: Go statik analiz
vet:
	$(GO) vet ./...

## tidy: Bağımlılıkları düzenle
tidy:
	$(GO) mod tidy

## clean: Derleme çıktılarını temizle
clean:
	rm -f $(BINARY)

## help: Bu yardım mesajını göster
help:
	@echo "promptTranslate — Makefile hedefleri"
	@echo "======================================"
	@grep -E '^## ' Makefile | sed 's/## /  make /'
