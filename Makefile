.PHONY: build test fmt vet run-demo

build:
	@go build -o zt-sim .

test:
	go test ./... -count=1

fmt:
	gofmt -w .

vet:
	go vet ./...

# Reproduces the README demo transcript from a clean build.
# decide exits 3 (Block) on purpose — `; echo $$?` shows it without failing make.
run-demo: build
	@rm -f audit.jsonl
	@./zt-sim decide --audit audit.jsonl examples/attacker-signals.json; echo $$?
	@./zt-sim run scenarios/; echo $$?
