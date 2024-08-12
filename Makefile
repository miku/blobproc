SHELL := /bin/bash
TARGETS := blobprocd # blobproc, TODO: add this
PKGNAME := blobproc

.PHONY: all
all: $(TARGETS)

%: cmd/%/main.go
	# GLIBC version mismatch on deployment target, use CGO_ENABLED=0
	CGO_ENABLED=0 go build -o $@ $<

.PHONY: test
test:
	go test -v -cover ./...

.PHONY: clean
clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)_*.deb


.PHONY: update-all-deps
update-all-deps:
	go get -u -v ./... && go mod tidy

.PHONY: deb
deb: $(TARGETS)
	mkdir -p packaging/deb/$(PKGNAME)/usr/local/bin
	cp $(TARGETS) packaging/deb/$(PKGNAME)/usr/local/bin
	cd packaging/deb && fakeroot dpkg-deb --build $(PKGNAME) .
	mv packaging/deb/$(PKGNAME)_*.deb .


