BINARY  := slack-status
DESTDIR := /usr/local/bin

-include .env
export

OAUTH_URL := https://api.slack.com/apps/$(APP_ID)/oauth
LDFLAGS := -X 'main.oauthURL="$(OAUTH_URL)"'

.PHONY: help init env build install uninstall

help:
	@echo ""
	@echo "To initialize, run: make init"
	@echo ""
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  init     Initialize .env file"
	@echo "  build    Build the binary"
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

install: build
	sudo install -m 755 $(BINARY) $(DESTDIR)/$(BINARY)

uninstall:
	sudo rm -f $(DESTDIR)/$(BINARY)
