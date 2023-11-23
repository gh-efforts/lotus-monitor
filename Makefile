 SHELL=/usr/bin/env bash

 all: build
.PHONY: all

unexport GOFLAGS

ldflags=-X=github.com/gh-efforts/lotus-monitor/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

build: lotus-monitor
.PHONY: build

calibnet: GOFLAGS+=-tags=calibnet
calibnet: build

lotus-monitor:
	rm -f lotus-monitor
	go build $(GOFLAGS) -o lotus-monitor ./cmd/lotus-monitor
.PHONY: lotus-monitor

clean:
	rm -f lotus-monitor
	go clean
.PHONY: clean