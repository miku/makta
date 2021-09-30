SHELL := /bin/bash
TARGETS := slikv
VERSION := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.Buildtime=$(BUILDTIME)
GOLDFLAGS += -w -s
GOFLAGS = -ldflags "$(GOLDFLAGS)"

.PHONY: all
all: $(TARGETS)

%: cmd/%/main.go
	go build -ldflags "$(GOLDFLAGS)" -o $@ $^

.PHONY: clean
clean:
	rm -f $(TARGETS)

.PHONY: purge
purge:
	rm -if data.db
