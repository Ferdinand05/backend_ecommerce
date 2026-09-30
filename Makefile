# 1. Membaca file .env
-include .env

# 2. Menyusun DB_URL dari variabel-variabel .env
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

MIGRATE_PATH=database/migrations

migrate-create:
	migrate create -ext sql -dir $(MIGRATE_PATH) -seq $(name)

migrate-up:
	migrate -path $(MIGRATE_PATH) -database "$(DB_URL)" -verbose up

migrate-down:
	migrate -path $(MIGRATE_PATH) -database "$(DB_URL)" -verbose down 1
