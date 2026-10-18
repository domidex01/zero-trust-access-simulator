.PHONY: build test fmt vet run-demo

build:
	go build -o zt-sim .

test:
	go test ./... -count=1

fmt:
	gofmt -w .

vet:
	go vet ./...

# Phase 0 stub; Phase 7 (T7.4) expands to the full §5 demo transcript.
run-demo: build
	./zt-sim --version
