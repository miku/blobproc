# BLOBPROC makefile
#
# requires: nfpm, pandoc, go

SHELL := /bin/bash
TARGETS := blobproc blobfetch docs/blobproc.1
PKGNAME := blobproc
MAKEFLAGS := --jobs=$(shell nproc)
VERSION := 0.3.33 # change this and then run "make update-version"

.PHONY: all
all: $(TARGETS)

%: cmd/%/main.go
	CGO_ENABLED=0 go build -o $@ $< # GLIBC version mismatch on deployment target, use CGO_ENABLED=0

.PHONY: test
test:
	go test -short -v -cover ./...

.PHONY: bench
bench:
	go test -short -bench=.

.PHONY: cover
cover:
	# may take 1m46.456s or longer
	go test -v -coverprofile=coverage.out -cover ./...
	go tool cover -html=coverage.out -o coverage.html
	# open coverage.html

docs/blobproc.1: docs/blobproc.md
	pandoc -s -t man docs/blobproc.md -o docs/blobproc.1

.PHONY: clean
clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)_*.deb
	rm -f coverage.out
	rm -f coverage.html
	rm -f docs/blobproc.1


.PHONY: update-all-deps
update-all-deps:
	go get -u -v ./... && go mod tidy

.PHONY: deb
deb: $(TARGETS)
	GOARCH=amd64 SEMVER=$(VERSION) nfpm package -c nfpm.yaml -p deb
	GOARCH=arm64 SEMVER=$(VERSION) nfpm package -c nfpm.yaml -p deb

.PHONY: update-version
update-version:
	sed -i -e 's@^const Version =.*@const Version = "$(VERSION)"@' version.go

