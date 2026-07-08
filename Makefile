MIGRATIONS_DIR := backend/internal/migrate/migrations
DATABASE_URL ?= postgres://werewolf:werewolf@localhost:5432/werewolf?sslmode=disable

.PHONY: up down build logs ps restart clean migrate-create migrate-up migrate-down

up:
	docker compose up -d --build

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f

ps:
	docker compose ps

restart: down up

clean:
	docker compose down -v

# backend applies pending migrations automatically on startup;
# these targets are for local convenience (scaffolding files / manual rollback).
migrate-create:
	docker run --rm -v $(PWD)/$(MIGRATIONS_DIR):/migrations migrate/migrate create -ext sql -dir /migrations -seq $(name)

migrate-up:
	docker run --rm --network host -v $(PWD)/$(MIGRATIONS_DIR):/migrations migrate/migrate -path=/migrations -database "$(DATABASE_URL)" up

migrate-down:
	docker run --rm --network host -v $(PWD)/$(MIGRATIONS_DIR):/migrations migrate/migrate -path=/migrations -database "$(DATABASE_URL)" down 1
