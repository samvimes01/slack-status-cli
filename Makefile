BINARY  := slack-status
DESTDIR := /usr/local/bin
MACOS_PROJECT := macos/SlackStatusApp.xcodeproj
MACOS_SCHEME := SlackStatusApp
MACOS_CONFIGURATION ?= Debug
MACOS_DERIVED_DATA := macos/build
MACOS_APP := $(MACOS_DERIVED_DATA)/Build/Products/$(MACOS_CONFIGURATION)/$(MACOS_SCHEME).app

-include .env
export

OAUTH_URL := https://api.slack.com/apps/$(APP_ID)/oauth
LDFLAGS := -X 'main.oauthURL="$(OAUTH_URL)"'

.PHONY: help init env build install uninstall macos-build macos-open macos-run

help:
	@echo ""
	@echo "To initialize, run: make init"
	@echo ""
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  init     Initialize .env file"
	@echo "  build    Build the binary"
	@echo "  macos-build Build the macOS menu bar app with xcodebuild"
	@echo "  macos-open  Launch the built macOS menu bar app via open"
	@echo "  macos-run   Build and run the macOS menu bar app directly"
	@echo "  install  Install the binary to $(DESTDIR)"
	@echo "  uninstall Uninstall the binary from $(DESTDIR)"
	@echo ""

init:
	@cp .env.example .env
	@echo ""
	@echo "Please edit .env to set APP_ID"
	@echo "Visit https://api.slack.com/apps and select \"status service\" app."
	@echo "Copy the app id from the URL <https://api.slack.com/apps/APP_ID/> and paste it into .env"
	@echo ""

env:
	@echo "APP_ID=$(APP_ID)"
	@echo "OAUTH_URL=$(OAUTH_URL)"
	@echo "LDFLAGS=$(LDFLAGS)"

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)

macos-build:
	xcodebuild -project $(MACOS_PROJECT) -scheme $(MACOS_SCHEME) -configuration $(MACOS_CONFIGURATION) -derivedDataPath $(MACOS_DERIVED_DATA) build

macos-open: macos-build
	open $(MACOS_APP)

macos-run: macos-build
	$(MACOS_APP)/Contents/MacOS/$(MACOS_SCHEME)

install: build
	sudo install -m 755 $(BINARY) $(DESTDIR)/$(BINARY)

uninstall:
	sudo rm -f $(DESTDIR)/$(BINARY)
