.PHONY: bootstrap infra-up infra-down api-dev api-test api-build web-dev web-lint web-build test build

bootstrap:
	npm install
	cd apps/api && go mod download

infra-up:
	docker compose up -d postgres minio minio-init

infra-down:
	docker compose down

api-dev:
	cd apps/api && go run ./cmd/api

api-test:
	cd apps/api && go test ./...

api-build:
	mkdir -p bin
	cd apps/api && go build -o ../../bin/api ./cmd/api

web-dev:
	npm run dev --workspace=@control-propiedades/web

web-lint:
	npm run lint --workspace=@control-propiedades/web

web-build:
	npm run build --workspace=@control-propiedades/web

test: api-test web-lint

build: api-build web-build
