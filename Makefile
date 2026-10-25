.PHONY: build test fmt vet run-demo screenshots

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

# Regenerate the README's terminal screenshots (docs/*.svg) from live output.
# decide's exit 3 (Block) is swallowed by the pipe — make sees python's 0.
# python3 is doc tooling only; the binary stays stdlib-only Go.
screenshots: build
	@rm -f /tmp/zt-ss-audit.jsonl
	@./zt-sim decide --audit /tmp/zt-ss-audit.jsonl examples/attacker-signals.json | python3 docs/term2svg.py "zt-sim decide — attacker-signals.json" > docs/decide.svg
	@./zt-sim run scenarios/ | python3 docs/term2svg.py "zt-sim run scenarios/" > docs/run.svg
	@./zt-sim policies | python3 docs/term2svg.py "zt-sim policies" > docs/policies.svg
	@./zt-sim audit /tmp/zt-ss-audit.jsonl | python3 docs/term2svg.py "zt-sim audit audit.jsonl" > docs/audit.svg
	@rm -f /tmp/zt-ss-audit.jsonl
