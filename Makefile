SHELL := /bin/bash
TARGETS := blobprocd blobproc
PKGNAME := blobproc
MAKEFLAGS := --jobs=$(shell nproc)
VERSION := 0.3.26 # change this and then run "make update-version"

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

.PHONY: clean
clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)_*.deb
	rm -f coverage.out
	rm -f coverage.html


.PHONY: update-all-deps
update-all-deps:
	go get -u -v ./... && go mod tidy

.PHONY: deb
deb: $(TARGETS)
	mkdir -p packaging/deb/$(PKGNAME)/usr/local/bin
	cp $(TARGETS) packaging/deb/$(PKGNAME)/usr/local/bin
	cd packaging/deb && fakeroot dpkg-deb --build $(PKGNAME) .
	mv packaging/deb/$(PKGNAME)_*.deb .


.PHONY: update-version
update-version:
	sed -i -e 's@^const Version =.*@const Version = "$(VERSION)"@' version.go
	sed -i -e 's@^Version:.*@Version: $(VERSION)@' packaging/deb/blobproc/DEBIAN/control

