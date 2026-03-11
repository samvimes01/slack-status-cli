BINARY  := slack-status
DESTDIR := /usr/local/bin
MACOS_PROJECT := macos/SlackStatusApp.xcodeproj
MACOS_SCHEME := SlackStatusApp
MACOS_CONFIGURATION ?= Debug
MACOS_DERIVED_DATA := macos/build
MACOS_APP := $(MACOS_DERIVED_DATA)/Build/Products/$(MACOS_CONFIGURATION)/$(MACOS_SCHEME).app

# Distribution variables
CLI_UNIVERSAL    := slack-status-universal
MACOS_DIST_APP   := $(MACOS_DERIVED_DATA)/Build/Products/Release/$(MACOS_SCHEME).app
MACOS_DIST_DIR   := dist
DMG_NAME         := SlackStatusApp.dmg
CODESIGN_IDENTITY ?= -

-include .env
export

OAUTH_URL := https://api.slack.com/apps/$(APP_ID)/oauth
LDFLAGS := -X 'main.oauthURL="$(OAUTH_URL)"'

.PHONY: help init env build install uninstall \
	macos-build macos-open macos-run \
	build-universal macos-build-release macos-bundle macos-sign \
	macos-stage macos-dmg macos-dist macos-clean

help:
	@echo ""
	@echo "To initialize, run: make init"
	@echo ""
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  init              Initialize .env file"
	@echo "  build             Build the binary"
	@echo "  install           Install the binary to $(DESTDIR)"
	@echo "  uninstall         Uninstall the binary from $(DESTDIR)"
	@echo "  macos-build       Build the macOS menu bar app with xcodebuild"
	@echo "  macos-open        Launch the built macOS menu bar app via open"
	@echo "  macos-run         Build and run the macOS menu bar app directly"
	@echo ""
	@echo "Distribution targets:"
	@echo "  macos-dist        Full pipeline: universal CLI + Release app + bundle + sign + DMG"
	@echo "  build-universal   Build a universal (amd64+arm64) Go CLI binary"
	@echo "  macos-build-release  Build the macOS app in Release configuration"
	@echo "  macos-bundle      Copy the universal CLI into the .app bundle"
	@echo "  macos-sign        Ad-hoc codesign the .app (override CODESIGN_IDENTITY for Developer ID)"
	@echo "  macos-stage       Stage the signed .app into dist/"
	@echo "  macos-dmg         Create SlackStatusApp.dmg from staged artifacts"
	@echo "  macos-clean       Remove all build and distribution artifacts"
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

# --- Distribution targets ---

build-universal:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-arm64
	lipo -create -output $(CLI_UNIVERSAL) $(BINARY)-amd64 $(BINARY)-arm64
	rm -f $(BINARY)-amd64 $(BINARY)-arm64

macos-build-release:
	xcodebuild -project $(MACOS_PROJECT) -scheme $(MACOS_SCHEME) \
		-configuration Release -derivedDataPath $(MACOS_DERIVED_DATA) build

macos-bundle: build-universal macos-build-release
	cp $(CLI_UNIVERSAL) "$(MACOS_DIST_APP)/Contents/Resources/slack-status"
	chmod 755 "$(MACOS_DIST_APP)/Contents/Resources/slack-status"

macos-sign: macos-bundle
	codesign --force --deep --sign "$(CODESIGN_IDENTITY)" "$(MACOS_DIST_APP)"
	codesign --verify --verbose "$(MACOS_DIST_APP)"

macos-stage: macos-sign
	rm -rf $(MACOS_DIST_DIR)
	mkdir -p $(MACOS_DIST_DIR)
	cp -R "$(MACOS_DIST_APP)" "$(MACOS_DIST_DIR)/"

macos-dmg: macos-stage
	rm -f $(DMG_NAME)
	hdiutil create -volname "SlackStatusApp" \
		-srcfolder $(MACOS_DIST_DIR) \
		-ov -format UDZO \
		$(DMG_NAME)
	@echo ""
	@echo "Distribution DMG created: $(DMG_NAME)"
	@echo "Recipients: right-click the app → Open on first launch."
	@echo ""

macos-dist: macos-dmg

macos-clean:
	rm -rf $(MACOS_DERIVED_DATA) $(MACOS_DIST_DIR) $(DMG_NAME) $(CLI_UNIVERSAL)
