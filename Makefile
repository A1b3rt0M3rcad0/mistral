.PHONY: test vet validate-content quality api frontend-install frontend-build

test:
	go test ./...

vet:
	go vet ./...

validate-content:
	go run ./tooling/content-validator -content ./content

quality:
	go run ./tooling/quality-gate -root .

api:
	go run ./packages/mistral-api/cmd/mistral-api -content ./content

frontend-install:
	cd packages/mistral-frontend && npm install

frontend-build:
	cd packages/mistral-frontend && npm run build
