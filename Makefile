 SHELL=/usr/bin/env bash

 all: build
.PHONY: all

unexport GOFLAGS

ldflags=-X=main.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

build:
	rm -f lotus-monitor
	go build $(GOFLAGS) -o lotus-monitor

clean:
	rm -f lotus-monitor
	go clean