PLUGIN_ID := $(shell sed -n 's/^ *"id": *"\([^"]*\)".*/\1/p' plugin.json | head -1)
VERSION   := $(shell sed -n 's/^ *"version": *"\([^"]*\)".*/\1/p' plugin.json | head -1)
BUNDLE    := dist/$(PLUGIN_ID)-$(VERSION).tar.gz
STAGE     := build/$(PLUGIN_ID)
GOFLAGS_BUILD := -trimpath -ldflags="-s -w"

.PHONY: all test server dist clean

all: test dist

## test: run unit tests
test:
	cd server && go vet ./... && go test -count=1 ./...

## server: build the Linux server binaries
server:
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS_BUILD) -o dist/plugin-linux-amd64 .
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS_BUILD) -o dist/plugin-linux-arm64 .

## dist: build the plugin bundle for upload in the System Console
# Mattermost's extractor needs explicit directory entries in the archive,
# so the members are listed one by one instead of archiving a folder.
dist: server
	rm -rf build && mkdir -p $(STAGE)/server/dist dist
	cp plugin.json $(STAGE)/
	cp server/dist/plugin-linux-amd64 server/dist/plugin-linux-arm64 $(STAGE)/server/dist/
	cd build && tar -czf ../$(BUNDLE) --no-recursion \
		$(PLUGIN_ID)/ $(PLUGIN_ID)/server/ $(PLUGIN_ID)/server/dist/ \
		$(PLUGIN_ID)/server/dist/plugin-linux-amd64 \
		$(PLUGIN_ID)/server/dist/plugin-linux-arm64 \
		$(PLUGIN_ID)/plugin.json
	@echo "Bundle: $(BUNDLE)"

clean:
	rm -rf build dist server/dist
