.PHONY: bootstrap infra-up infra-down migrate-up migrate-down migrate-status api-dev api-test api-build web-dev web-lint web-build test build

bootstrap:
	npm install
	cd apps/api && go mod download

infra-up:
	docker compose up -d postgres minio minio-init

infra-down:
	docker compose down

migrate-up:
	set -a; . ./.env; set +a; cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ../../migrations postgres "$$DATABASE_URL" up

migrate-down:
	set -a; . ./.env; set +a; cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ../../migrations postgres "$$DATABASE_URL" down

migrate-status:
	set -a; . ./.env; set +a; cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ../../migrations postgres "$$DATABASE_URL" status

api-dev:
	set -a; . ./.env; set +a; cd apps/api && go run ./cmd/api

api-test:
	cd apps/api && go test ./...

api-build:
	mkdir -p bin
	cd apps/api && go build -o ../../bin/api ./cmd/api

web-dev:
	set -a; . ./.env; set +a; npm run dev --workspace=@control-propiedades/web

web-lint:
	npm run lint --workspace=@control-propiedades/web

web-build:
	npm run build --workspace=@control-propiedades/web

test: api-test web-lint

build: api-build web-build
