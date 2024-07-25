SHELL := /bin/bash
TARGETS := blobrun
.PHONY: all
all: $(TARGETS)

%: %.go
	go build -o $@ $<

.PHONY: clean
clean:
	rm -f $(TARGETS)

.PHONY: update-all-deps
update-all-deps:
	go get -u -v ./... && go mod tidy

