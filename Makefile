GO ?= go
NPM ?= npm

.PHONY: test vet validate-content quality-gate frontend-build check

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

validate-content:
	$(GO) run ./tooling/content-validator -content ./content

quality-gate:
	$(GO) run ./tooling/quality-gate -root .

frontend-build:
	cd packages/mistral-frontend && $(NPM) install && $(NPM) run build

check: test vet validate-content quality-gate
